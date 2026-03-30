package venom

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// appConfig holds configuration for viper and cobra setup.
type appConfig struct {
	appName     string
	envPrefix   string
	version     string
	configPaths []string
	configName  string
}

// setupViper initializes the global viper instance with the given app config.
func setupViper(cfg *appConfig) error {
	// Env prefix: use explicit envPrefix, or SCREAMING_SNAKE of appName.
	prefix := cfg.envPrefix
	if prefix == "" {
		prefix = strings.ToUpper(strings.ReplaceAll(cfg.appName, "-", "_"))
	}
	viper.SetEnvPrefix(prefix)

	viper.AutomaticEnv()

	// Replace hyphens with underscores so kebab-case flags match SCREAMING_SNAKE env vars.
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Config file name.
	configName := cfg.configName
	if configName == "" {
		configName = "." + cfg.appName
	}
	viper.SetConfigName(configName)

	// Config search paths.
	paths := cfg.configPaths
	if len(paths) == 0 {
		paths = []string{".", "$HOME"}
	}
	for _, p := range paths {
		viper.AddConfigPath(p)
	}

	// Read config file; ignore "not found" errors.
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return err
		}
	}
	return nil
}

// bindCommandFlags binds flags and env vars for the currently executing command.
// It finds the matching FuncMeta by comparing command paths, then binds each
// parameter's flag and env var to the global viper instance.
func bindCommandFlags(cfg *appConfig, cmd *cobra.Command, metas []*FuncMeta) {
	// Build the command path from the executing command.
	cmdPath := buildCommandPath(cmd)

	// Find the matching FuncMeta.
	var meta *FuncMeta
	for _, m := range metas {
		if pathsEqual(m.CommandPath, cmdPath) {
			meta = m
			break
		}
	}
	if meta == nil {
		return
	}

	// Derive env prefix.
	prefix := cfg.envPrefix
	if prefix == "" {
		prefix = strings.ToUpper(strings.ReplaceAll(cfg.appName, "-", "_"))
	}

	for _, p := range meta.Params {
		configKey := flagToConfigKey(p.FlagName)
		envVar := flagToEnvVar(prefix, p.FlagName)

		if f := cmd.Flags().Lookup(p.FlagName); f != nil {
			_ = viper.BindPFlag(configKey, f)
		}
		_ = viper.BindEnv(configKey, envVar)
	}
}

// buildCommandPath returns the command names from root to cmd, excluding the
// root command itself.
func buildCommandPath(cmd *cobra.Command) []string {
	var path []string
	for c := cmd; c != nil; c = c.Parent() {
		path = append(path, c.Name())
	}
	// Reverse and drop root.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	if len(path) > 0 {
		path = path[1:] // drop root command
	}
	return path
}

// pathsEqual returns true if two string slices are identical.
func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
