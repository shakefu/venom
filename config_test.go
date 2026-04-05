package venom

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestSetupViperDefaults(t *testing.T) {
	viper.Reset()

	cfg := &appConfig{
		appName: "myapp",
	}
	if err := setupViper(cfg); err != nil {
		t.Fatalf("setupViper returned error: %v", err)
	}

	// Env prefix should be derived as SCREAMING_SNAKE of appName.
	// Verify by setting an env var and reading it through viper.
	t.Run("env_prefix_derived_from_app_name", func(t *testing.T) {
		viper.Reset()
		cfg := &appConfig{appName: "my-app"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper returned error: %v", err)
		}
		os.Setenv("MY_APP_SOME_KEY", "hello")
		defer os.Unsetenv("MY_APP_SOME_KEY")
		// AutomaticEnv + prefix means viper.Get("some_key") should resolve.
		got := viper.GetString("some_key")
		if got != "hello" {
			t.Errorf("expected env prefix MY_APP to resolve some_key = %q, got %q", "hello", got)
		}
	})

	t.Run("config_name_defaults_to_dot_appname", func(t *testing.T) {
		viper.Reset()
		cfg := &appConfig{appName: "myapp"}
		// We can't directly inspect viper's config name, but we verify no error
		// is returned and the function completes (config file not found is OK).
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper returned error: %v", err)
		}
	})

	t.Run("default_config_paths", func(t *testing.T) {
		// The spec says default paths are {".", "$HOME"}.
		// We verify this indirectly: when configPaths is empty, setupViper
		// should add "." and "$HOME". Since there is no config file in either
		// location, ReadInConfig returns ConfigFileNotFoundError which is
		// swallowed, so setupViper should succeed.
		viper.Reset()
		cfg := &appConfig{appName: "myapp"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper returned error: %v", err)
		}
	})
}

func TestSetupViperExplicitConfig(t *testing.T) {
	viper.Reset()

	cfg := &appConfig{
		appName:     "myapp",
		envPrefix:   "CUSTOM_PREFIX",
		configName:  "custom-config",
		configPaths: []string{"/tmp"},
	}
	if err := setupViper(cfg); err != nil {
		t.Fatalf("setupViper returned error: %v", err)
	}

	// Verify explicit env prefix is used.
	os.Setenv("CUSTOM_PREFIX_TEST_VAR", "works")
	defer os.Unsetenv("CUSTOM_PREFIX_TEST_VAR")

	got := viper.GetString("test_var")
	if got != "works" {
		t.Errorf("expected explicit env prefix to resolve test_var = %q, got %q", "works", got)
	}
}

func TestBuildCommandPath(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *cobra.Command
		expected []string
	}{
		{
			name: "single_subcommand",
			setup: func() *cobra.Command {
				root := &cobra.Command{Use: "app"}
				child := &cobra.Command{Use: "serve"}
				root.AddCommand(child)
				return child
			},
			expected: []string{"serve"},
		},
		{
			name: "nested_subcommands",
			setup: func() *cobra.Command {
				root := &cobra.Command{Use: "app"}
				parent := &cobra.Command{Use: "init"}
				child := &cobra.Command{Use: "project"}
				root.AddCommand(parent)
				parent.AddCommand(child)
				return child
			},
			expected: []string{"init", "project"},
		},
		{
			name: "root_command_returns_empty",
			setup: func() *cobra.Command {
				root := &cobra.Command{Use: "app"}
				return root
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.setup()
			got := buildCommandPath(cmd)
			if !pathsEqual(got, tt.expected) {
				t.Errorf("buildCommandPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPathsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []string
		expected bool
	}{
		{"equal", []string{"a", "b"}, []string{"a", "b"}, true},
		{"unequal_values", []string{"a", "b"}, []string{"a", "c"}, false},
		{"unequal_lengths", []string{"a"}, []string{"a", "b"}, false},
		{"both_empty", []string{}, []string{}, true},
		{"both_nil", nil, nil, true},
		{"nil_vs_empty", nil, []string{}, true},
		{"one_nil_one_populated", nil, []string{"a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathsEqual(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("pathsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestBindCommandFlags(t *testing.T) {
	viper.Reset()

	cfg := &appConfig{
		appName: "testapp",
	}

	// Set up a cobra command with a flag.
	root := &cobra.Command{Use: "testapp"}
	child := &cobra.Command{Use: "serve"}
	child.Flags().String("port", "8080", "server port")
	root.AddCommand(child)

	metas := []*FuncMeta{
		{
			FullName:    "main.serve",
			CommandPath: []string{"serve"},
			Params: []ParamMeta{
				{Name: "port", Type: "string", FlagName: "port", Default: "8080", Desc: "server port"},
			},
		},
	}

	// Set the flag value.
	child.Flags().Set("port", "9090")

	bindCommandFlags(cfg, child, metas)

	// Viper should now be able to read the flag value.
	got := viper.GetString("port")
	if got != "9090" {
		t.Errorf("expected viper to read flag value %q, got %q", "9090", got)
	}
}

func TestResolveParamValuePriority(t *testing.T) {
	t.Run("flag_wins_over_env", func(t *testing.T) {
		viper.Reset()

		cfg := &appConfig{appName: "testapp"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper: %v", err)
		}

		root := &cobra.Command{Use: "testapp"}
		child := &cobra.Command{Use: "serve"}
		child.Flags().String("host", "", "server host")
		root.AddCommand(child)

		metas := []*FuncMeta{
			{
				FullName:    "main.serve",
				CommandPath: []string{"serve"},
				Params: []ParamMeta{
					{Name: "host", Type: "string", FlagName: "host"},
				},
			},
		}

		// Set env value.
		os.Setenv("TESTAPP_HOST", "env-host")
		defer os.Unsetenv("TESTAPP_HOST")

		// Set flag value (simulates CLI --host=flag-host).
		child.Flags().Set("host", "flag-host")

		bindCommandFlags(cfg, child, metas)

		got := viper.GetString("host")
		if got != "flag-host" {
			t.Errorf("expected flag to win: got %q, want %q", got, "flag-host")
		}
	})

	t.Run("env_wins_when_flag_not_set", func(t *testing.T) {
		viper.Reset()

		cfg := &appConfig{appName: "testapp"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper: %v", err)
		}

		root := &cobra.Command{Use: "testapp"}
		child := &cobra.Command{Use: "serve"}
		child.Flags().String("host", "", "server host")
		root.AddCommand(child)

		metas := []*FuncMeta{
			{
				FullName:    "main.serve",
				CommandPath: []string{"serve"},
				Params: []ParamMeta{
					{Name: "host", Type: "string", FlagName: "host"},
				},
			},
		}

		os.Setenv("TESTAPP_HOST", "env-host")
		defer os.Unsetenv("TESTAPP_HOST")

		bindCommandFlags(cfg, child, metas)

		got := viper.GetString("host")
		if got != "env-host" {
			t.Errorf("expected env to win: got %q, want %q", got, "env-host")
		}
	})

	t.Run("default_applies_when_nothing_set", func(t *testing.T) {
		viper.Reset()

		cfg := &appConfig{appName: "testapp"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper: %v", err)
		}

		root := &cobra.Command{Use: "testapp"}
		child := &cobra.Command{Use: "serve"}
		child.Flags().String("host", "default-host", "server host")
		root.AddCommand(child)

		metas := []*FuncMeta{
			{
				FullName:    "main.serve",
				CommandPath: []string{"serve"},
				Params: []ParamMeta{
					{Name: "host", Type: "string", FlagName: "host", Default: "default-host"},
				},
			},
		}

		bindCommandFlags(cfg, child, metas)

		got := viper.GetString("host")
		if got != "default-host" {
			t.Errorf("expected default: got %q, want %q", got, "default-host")
		}
	})
}

func TestRequiredFlagValidation(t *testing.T) {
	t.Run("required_flag_not_set_returns_error", func(t *testing.T) {
		viper.Reset()

		cfg := &appConfig{appName: "testapp"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper: %v", err)
		}

		// A dummy function that matches the expected signature: (context.Context, string) error.
		fn := func(ctx context.Context, port string) error {
			return nil
		}

		meta := &FuncMeta{
			FullName:    "main.serve",
			CommandPath: []string{"serve"},
			Params: []ParamMeta{
				{Name: "port", Type: "string", FlagName: "port", Required: true, Desc: "server port"},
			},
			Func: fn,
		}

		cmd := buildCommand(meta)
		root := &cobra.Command{Use: "testapp"}
		root.AddCommand(cmd)

		// Bind flags so viper knows about them.
		bindCommandFlags(cfg, cmd, []*FuncMeta{meta})

		// Execute the RunE directly without setting the flag.
		err := cmd.RunE(cmd, []string{})
		if err == nil {
			t.Fatal("expected error for required flag not set, got nil")
		}
		expected := `required flag "port" not set`
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("required_flag_satisfied_by_env", func(t *testing.T) {
		viper.Reset()

		cfg := &appConfig{appName: "testapp"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper: %v", err)
		}

		fn := func(ctx context.Context, port string) error {
			return nil
		}

		meta := &FuncMeta{
			FullName:    "main.serve",
			CommandPath: []string{"serve"},
			Params: []ParamMeta{
				{Name: "port", Type: "string", FlagName: "port", Required: true, Desc: "server port"},
			},
			Func: fn,
		}

		cmd := buildCommand(meta)
		root := &cobra.Command{Use: "testapp"}
		root.AddCommand(cmd)

		// Set env var to satisfy the required flag.
		os.Setenv("TESTAPP_PORT", "3000")
		defer os.Unsetenv("TESTAPP_PORT")

		bindCommandFlags(cfg, cmd, []*FuncMeta{meta})

		err := cmd.RunE(cmd, []string{})
		if err != nil {
			t.Fatalf("expected no error when required flag satisfied by env, got: %v", err)
		}
	})

	t.Run("required_flag_satisfied_by_flag", func(t *testing.T) {
		viper.Reset()

		cfg := &appConfig{appName: "testapp"}
		if err := setupViper(cfg); err != nil {
			t.Fatalf("setupViper: %v", err)
		}

		meta := &FuncMeta{
			FullName:    "main.serve",
			CommandPath: []string{"serve"},
			Params: []ParamMeta{
				{Name: "port", Type: "string", FlagName: "port", Required: true, Desc: "server port"},
			},
			Func: func(ctx context.Context, port string) error {
				return nil
			},
		}

		cmd := buildCommand(meta)
		root := &cobra.Command{Use: "testapp"}
		root.AddCommand(cmd)

		// Set the flag directly.
		cmd.Flags().Set("port", "8080")

		bindCommandFlags(cfg, cmd, []*FuncMeta{meta})

		err := cmd.RunE(cmd, []string{})
		if err != nil {
			t.Fatalf("expected no error when required flag set via CLI, got: %v", err)
		}
	})
}
