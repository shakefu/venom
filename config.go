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
	extensions  []extension
}

// setupViper configures the given viper instance with the env prefix, config
// name, and config search paths derived from cfg, then attempts to read the
// config file. A "config file not found" error is swallowed; any other error
// is returned.
func setupViper(cfg *appConfig, v *viper.Viper) error {
	// Env prefix: use explicit envPrefix, or SCREAMING_SNAKE of appName.
	prefix := cfg.envPrefix
	if prefix == "" {
		prefix = strings.ToUpper(strings.ReplaceAll(cfg.appName, "-", "_"))
	}
	v.SetEnvPrefix(prefix)

	v.AutomaticEnv()

	// Replace hyphens with underscores so kebab-case flags match SCREAMING_SNAKE env vars.
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Config file name.
	configName := cfg.configName
	if configName == "" {
		configName = "." + cfg.appName
	}
	v.SetConfigName(configName)

	// Config search paths.
	paths := cfg.configPaths
	if len(paths) == 0 {
		paths = []string{".", "$HOME"}
	}
	for _, p := range paths {
		v.AddConfigPath(p)
	}

	// Read config file; ignore "not found" errors.
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return err
		}
	}
	return nil
}

// bindCommandFlags binds flags and env vars for the currently executing
// command to the given viper instance. It finds the matching FuncMeta by
// comparing command paths, then binds each parameter's flag and env var.
func bindCommandFlags(cfg *appConfig, v *viper.Viper, cmd *cobra.Command, metas []*FuncMeta) {
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
			_ = v.BindPFlag(configKey, f)
		}
		_ = v.BindEnv(configKey, envVar)
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
