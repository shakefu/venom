package venom

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type testExitError struct {
	code int
	msg  string
}

func (e *testExitError) Error() string  { return e.msg }
func (e *testExitError) ErrorCode() int { return e.code }

// testNoopCmd is a command function that succeeds.
func testNoopCmd(ctx context.Context) error { return nil }

// testErrorCmd is a command function that returns a plain error.
func testErrorCmd(ctx context.Context) error { return fmt.Errorf("boom") }

// testExitCodeCmd returns an error implementing ExitCoder.
func testExitCodeCmd(ctx context.Context) error {
	return &testExitError{code: 42, msg: "custom exit"}
}

// testParamCmd accepts typed parameters so we can exercise flag resolution.
func testParamCmd(ctx context.Context, name string, count int) error { return nil }

// runtimeName returns the runtime-qualified function name for fn.
func runtimeName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

// ---------------------------------------------------------------------------
// TestBuildCommandTree
// ---------------------------------------------------------------------------

func TestBuildCommandTree(t *testing.T) {
	metas := []*FuncMeta{
		{
			FullName:    runtimeName(testNoopCmd),
			CommandPath: []string{"serve"},
			Description: "Start the server",
			Func:        testNoopCmd,
		},
		{
			FullName:    runtimeName(testNoopCmd),
			CommandPath: []string{"serve", "tls"},
			Description: "Start TLS server",
			Func:        testNoopCmd,
		},
		{
			FullName:    runtimeName(testNoopCmd),
			CommandPath: []string{"init", "project"},
			Description: "Initialize a project",
			Func:        testNoopCmd,
		},
	}

	root := buildCommandTree("myapp", metas)

	t.Run("root Use is appName", func(t *testing.T) {
		if root.Use != "myapp" {
			t.Fatalf("expected root Use %q, got %q", "myapp", root.Use)
		}
	})

	t.Run("top-level commands", func(t *testing.T) {
		names := map[string]bool{}
		for _, c := range root.Commands() {
			names[c.Name()] = true
		}
		if !names["serve"] {
			t.Fatal("expected top-level command 'serve'")
		}
		if !names["init"] {
			t.Fatal("expected top-level command 'init'")
		}
	})

	t.Run("nested serve tls", func(t *testing.T) {
		serve, _, err := root.Find([]string{"serve"})
		if err != nil {
			t.Fatalf("find serve: %v", err)
		}
		var found bool
		for _, c := range serve.Commands() {
			if c.Name() == "tls" {
				found = true
				if c.Short != "Start TLS server" {
					t.Fatalf("expected Short %q, got %q", "Start TLS server", c.Short)
				}
			}
		}
		if !found {
			t.Fatal("expected nested command 'tls' under 'serve'")
		}
	})

	t.Run("nested init project", func(t *testing.T) {
		initCmd, _, err := root.Find([]string{"init"})
		if err != nil {
			t.Fatalf("find init: %v", err)
		}
		var found bool
		for _, c := range initCmd.Commands() {
			if c.Name() == "project" {
				found = true
			}
		}
		if !found {
			t.Fatal("expected nested command 'project' under 'init'")
		}
	})

	t.Run("intermediate group has no RunE", func(t *testing.T) {
		initCmd, _, _ := root.Find([]string{"init"})
		if initCmd.RunE != nil {
			t.Fatal("intermediate group 'init' should not have RunE")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAttachCommand
// ---------------------------------------------------------------------------

func TestAttachCommand(t *testing.T) {
	t.Run("single depth", func(t *testing.T) {
		root := &cobra.Command{Use: "app"}
		cmd := &cobra.Command{Use: "run"}
		attachCommand(root, []string{"run"}, cmd)

		if len(root.Commands()) != 1 {
			t.Fatalf("expected 1 child, got %d", len(root.Commands()))
		}
		if root.Commands()[0].Name() != "run" {
			t.Fatalf("expected child 'run', got %q", root.Commands()[0].Name())
		}
	})

	t.Run("multiple depth creates intermediates", func(t *testing.T) {
		root := &cobra.Command{Use: "app"}
		cmd := &cobra.Command{Use: "deep"}
		attachCommand(root, []string{"a", "b", "deep"}, cmd)

		a, _, _ := root.Find([]string{"a"})
		if a == nil {
			t.Fatal("expected intermediate 'a'")
		}
		var b *cobra.Command
		for _, c := range a.Commands() {
			if c.Name() == "b" {
				b = c
			}
		}
		if b == nil {
			t.Fatal("expected intermediate 'b' under 'a'")
		}
		var found bool
		for _, c := range b.Commands() {
			if c.Name() == "deep" {
				found = true
			}
		}
		if !found {
			t.Fatal("expected 'deep' under 'a b'")
		}
	})

	t.Run("reuses existing intermediate", func(t *testing.T) {
		root := &cobra.Command{Use: "app"}
		cmd1 := &cobra.Command{Use: "one"}
		cmd2 := &cobra.Command{Use: "two"}
		attachCommand(root, []string{"group", "one"}, cmd1)
		attachCommand(root, []string{"group", "two"}, cmd2)

		// Only one "group" intermediate should exist.
		groupCount := 0
		for _, c := range root.Commands() {
			if c.Name() == "group" {
				groupCount++
			}
		}
		if groupCount != 1 {
			t.Fatalf("expected 1 'group' intermediate, got %d", groupCount)
		}

		group, _, _ := root.Find([]string{"group"})
		if len(group.Commands()) != 2 {
			t.Fatalf("expected 2 children under 'group', got %d", len(group.Commands()))
		}
	})
}

// ---------------------------------------------------------------------------
// TestBuildCommand
// ---------------------------------------------------------------------------

func TestBuildCommand(t *testing.T) {
	meta := &FuncMeta{
		FullName:    runtimeName(testNoopCmd),
		CommandPath: []string{"serve", "tls"},
		Description: "Start TLS server",
		Func:        testNoopCmd,
	}

	cmd := buildCommand(meta)

	t.Run("Use is last path segment", func(t *testing.T) {
		if cmd.Use != "tls" {
			t.Fatalf("expected Use %q, got %q", "tls", cmd.Use)
		}
	})

	t.Run("Short is Description", func(t *testing.T) {
		if cmd.Short != "Start TLS server" {
			t.Fatalf("expected Short %q, got %q", "Start TLS server", cmd.Short)
		}
	})

	t.Run("RunE is set", func(t *testing.T) {
		if cmd.RunE == nil {
			t.Fatal("expected RunE to be non-nil")
		}
	})
}

// ---------------------------------------------------------------------------
// TestRegisterFlag
// ---------------------------------------------------------------------------

func TestRegisterFlag(t *testing.T) {
	types := []struct {
		name     string
		paramTyp string
		defVal   string
		short    string
	}{
		{"string", "string", "hello", "s"},
		{"string_no_short", "string", "", ""},
		{"int", "int", "42", "i"},
		{"int_no_short", "int", "0", ""},
		{"int64", "int64", "100", "l"},
		{"int64_no_short", "int64", "0", ""},
		{"float64", "float64", "3.14", "f"},
		{"float64_no_short", "float64", "0", ""},
		{"bool", "bool", "true", "b"},
		{"bool_no_short", "bool", "false", ""},
		{"string_slice", "[]string", "", "a"},
		{"string_slice_no_short", "[]string", "", ""},
		{"duration", "time.Duration", "5s", "d"},
		{"duration_no_short", "time.Duration", "", ""},
	}

	for _, tc := range types {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			p := ParamMeta{
				Name:     tc.name,
				Type:     tc.paramTyp,
				FlagName: tc.name,
				Short:    tc.short,
				Default:  tc.defVal,
				Desc:     "test flag",
			}
			registerFlag(cmd, p)

			flag := cmd.Flags().Lookup(tc.name)
			if flag == nil {
				t.Fatalf("expected flag %q to be registered", tc.name)
			}
			if tc.short != "" && flag.Shorthand != tc.short {
				t.Fatalf("expected shorthand %q, got %q", tc.short, flag.Shorthand)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMakeRunFunc
// ---------------------------------------------------------------------------

func TestMakeRunFunc(t *testing.T) {
	t.Run("nil return success", func(t *testing.T) {
		viper.Reset()

		meta := &FuncMeta{
			FullName:    runtimeName(testNoopCmd),
			CommandPath: []string{"noop"},
			Description: "no-op command",
			Func:        testNoopCmd,
		}

		runE := makeRunFunc(meta)
		cmd := &cobra.Command{Use: "noop"}
		cmd.SetContext(context.Background())

		err := runE(cmd, nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("error return", func(t *testing.T) {
		viper.Reset()

		meta := &FuncMeta{
			FullName:    runtimeName(testErrorCmd),
			CommandPath: []string{"fail"},
			Description: "failing command",
			Func:        testErrorCmd,
		}

		runE := makeRunFunc(meta)
		cmd := &cobra.Command{Use: "fail"}
		cmd.SetContext(context.Background())

		err := runE(cmd, nil)
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if err.Error() != "boom" {
			t.Fatalf("expected error %q, got %q", "boom", err.Error())
		}
	})

	t.Run("ExitCoder error return", func(t *testing.T) {
		viper.Reset()

		meta := &FuncMeta{
			FullName:    runtimeName(testExitCodeCmd),
			CommandPath: []string{"exitcode"},
			Description: "exit code command",
			Func:        testExitCodeCmd,
		}

		runE := makeRunFunc(meta)
		cmd := &cobra.Command{Use: "exitcode"}
		cmd.SetContext(context.Background())

		err := runE(cmd, nil)
		if err == nil {
			t.Fatal("expected non-nil error")
		}

		var ec ExitCoder
		ok := false
		if e, is := err.(ExitCoder); is {
			ec = e
			ok = true
		}
		if !ok {
			t.Fatal("expected error to implement ExitCoder")
		}
		if ec.ErrorCode() != 42 {
			t.Fatalf("expected exit code 42, got %d", ec.ErrorCode())
		}
	})

	t.Run("with parameters", func(t *testing.T) {
		viper.Reset()

		meta := &FuncMeta{
			FullName:    runtimeName(testParamCmd),
			CommandPath: []string{"param"},
			Description: "param command",
			Params: []ParamMeta{
				{Name: "name", Type: "string", FlagName: "name", Default: "default"},
				{Name: "count", Type: "int", FlagName: "count", Default: "5"},
			},
			Func: testParamCmd,
		}

		cmd := buildCommand(meta)
		cmd.SetContext(context.Background())

		// Bind flags to viper so makeRunFunc can read them.
		viper.BindPFlags(cmd.Flags())

		runE := makeRunFunc(meta)
		err := runE(cmd, nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("required flag missing", func(t *testing.T) {
		viper.Reset()

		meta := &FuncMeta{
			FullName:    runtimeName(testParamCmd),
			CommandPath: []string{"param"},
			Description: "param command",
			Params: []ParamMeta{
				{Name: "name", Type: "string", FlagName: "name", Required: true},
				{Name: "count", Type: "int", FlagName: "count", Default: "0"},
			},
			Func: testParamCmd,
		}

		cmd := buildCommand(meta)
		cmd.SetContext(context.Background())
		viper.BindPFlags(cmd.Flags())

		runE := makeRunFunc(meta)
		err := runE(cmd, nil)
		if err == nil {
			t.Fatal("expected error for missing required flag")
		}
	})
}

// ---------------------------------------------------------------------------
// TestIsZero
// ---------------------------------------------------------------------------

func TestIsZero(t *testing.T) {
	cases := []struct {
		name string
		val  interface{}
		want bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 1, false},
		{"zero int64", int64(0), true},
		{"non-zero int64", int64(1), false},
		{"zero float64", float64(0), true},
		{"non-zero float64", float64(3.14), false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"zero duration", time.Duration(0), true},
		{"non-zero duration", 5 * time.Second, false},
		{"nil slice", ([]string)(nil), true},
		{"empty slice", []string{}, false}, // empty but not nil
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isZero(tc.val)
			if got != tc.want {
				t.Fatalf("isZero(%v) = %v, want %v", tc.val, got, tc.want)
			}
		})
	}
}
