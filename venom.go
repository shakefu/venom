package venom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// Option configures an App before execution.
type Option func(*appConfig)

// WithName overrides the derived application name.
func WithName(name string) Option {
	return func(cfg *appConfig) {
		cfg.appName = name
	}
}

// WithEnvPrefix sets the environment variable prefix used by viper.
// When empty, the prefix is derived from the app name in SCREAMING_SNAKE_CASE.
func WithEnvPrefix(prefix string) Option {
	return func(cfg *appConfig) {
		cfg.envPrefix = prefix
	}
}

// WithVersion sets the version string reported by --version.
func WithVersion(version string) Option {
	return func(cfg *appConfig) {
		cfg.version = version
	}
}

// WithConfigName sets the config file name (without extension) that viper
// searches for. The default is ".<appName>".
func WithConfigName(name string) Option {
	return func(cfg *appConfig) {
		cfg.configName = name
	}
}

// WithConfigPaths sets the directories viper searches for a config file.
// The default is [".", "$HOME"].
func WithConfigPaths(paths ...string) Option {
	return func(cfg *appConfig) {
		cfg.configPaths = paths
	}
}

// App holds configuration and provides an Execute method for running the CLI.
type App struct {
	cfg appConfig
}

// New creates an App with the given options. Use App.Execute to run the CLI.
func New(opts ...Option) *App {
	a := &App{
		cfg: defaultConfig(),
	}
	for _, opt := range opts {
		opt(&a.cfg)
	}
	return a
}

// Execute resolves metadata for the given command functions and runs the CLI.
// On error it prints to stderr and calls os.Exit(1).
func (a *App) Execute(fns ...interface{}) {
	if err := a.run(context.Background(), fns); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code := 1
		if ec, ok := err.(ExitCoder); ok {
			code = ec.ErrorCode()
		}
		os.Exit(code)
	}
}

// run is the internal implementation shared by all public entry points.
func (a *App) run(ctx context.Context, fns []interface{}) error {
	metas, err := resolveMetas(fns)
	if err != nil {
		return err
	}

	if err := setupViper(&a.cfg); err != nil {
		return err
	}

	root := buildCommandTree(a.cfg.appName, metas)

	if a.cfg.version != "" {
		root.Version = a.cfg.version
	}

	// Bind viper to the executing command's flags just before each run.
	cfg := a.cfg // capture for closure
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		bindCommandFlags(&cfg, cmd, metas)
		return nil
	}

	// Build fang options for styled help, version, completions, man pages.
	var fangOpts []fang.Option
	if a.cfg.version != "" {
		fangOpts = append(fangOpts, fang.WithVersion(a.cfg.version))
	}

	return fang.Execute(ctx, root, fangOpts...)
}

// Execute is the simplest entry point: it derives defaults from the binary name
// and runs the CLI. On error it prints to stderr and calls os.Exit(1).
func Execute(fns ...interface{}) {
	New().Execute(fns...)
}

// defaultConfig returns an appConfig with sensible defaults.
func defaultConfig() appConfig {
	name := "app"
	if len(os.Args) > 0 {
		name = filepath.Base(os.Args[0])
	}
	return appConfig{
		appName: name,
	}
}

// resolveMetas looks up FuncMeta for each function value.
func resolveMetas(fns []interface{}) ([]*FuncMeta, error) {
	metas := make([]*FuncMeta, 0, len(fns))
	for _, fn := range fns {
		m, err := lookupMeta(fn)
		if err != nil {
			return nil, err
		}
		metas = append(metas, m)
	}
	return metas, nil
}
