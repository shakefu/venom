package venom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

// WithVersion sets the version string reported by --version. When unset,
// the version is resolved from runtime/debug.ReadBuildInfo, then VCS
// revision, then a build timestamp — see rule ResolveAppVersion in
// venom.allium.
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

// App holds configuration and provides Execute and Build methods. Each call
// to Execute or Build produces a fresh cobra tree and viper instance; the
// App itself is reusable.
type App struct {
	cfg appConfig
}

// New creates an App with the given options. Use App.Execute to run the CLI
// or App.Build to obtain the constructed artifacts for custom dispatch.
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
// On error it prints to stderr and calls os.Exit(1) (or the ExitCoder code).
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

// Build resolves metadata, configures viper, builds the cobra command tree,
// and applies any registered extensions, then returns the constructed cobra
// root and viper instance to the caller. Use this when you want to compose
// Venom's output with additional cobra/viper logic, customise fang
// invocation, or own the execution loop entirely.
//
// Each call produces a fresh cobra tree and viper instance.
//
// If any extension returns a non-nil error, Build returns nil artifacts and
// the extension's error.
func (a *App) Build(fns ...interface{}) (*cobra.Command, *viper.Viper, error) {
	root, v, _, err := a.prepare(fns)
	if err != nil {
		return nil, nil, err
	}
	return root, v, nil
}

// run is the internal Execute implementation. It shares the preparation
// pipeline with Build and dispatches via fang.
func (a *App) run(ctx context.Context, fns []interface{}) error {
	root, _, version, err := a.prepare(fns)
	if err != nil {
		return err
	}

	fangOpts := []fang.Option{fang.WithVersion(version)}
	return fang.Execute(ctx, root, fangOpts...)
}

// prepare runs the App preparation pipeline shared by Execute and Build:
// resolve metadata, create a fresh viper instance, configure it, build the
// cobra command tree, set the version, wire flag binding for the executing
// command, and apply registered extensions in declaration order.
//
// Returns the cobra root, viper instance, resolved version string, and any
// error produced by metadata resolution, viper setup, or an extension
// callback. On error all returned artifacts are nil.
func (a *App) prepare(fns []interface{}) (*cobra.Command, *viper.Viper, string, error) {
	metas, err := resolveMetas(fns)
	if err != nil {
		return nil, nil, "", err
	}

	v := viper.New()
	if err := setupViper(&a.cfg, v); err != nil {
		return nil, nil, "", err
	}

	root := buildCommandTree(a.cfg.appName, metas, v)

	version := effectiveVersion(a.cfg.version)
	root.Version = version

	// Bind viper to the executing command's flags just before each run.
	cfg := a.cfg // capture for closure
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		bindCommandFlags(&cfg, v, cmd, metas)
		return nil
	}

	if err := applyExtensions(a.cfg.extensions, root, v); err != nil {
		return nil, nil, "", err
	}

	return root, v, version, nil
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
