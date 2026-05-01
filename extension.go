package venom

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// extensionTarget identifies which underlying artifact an extension callback
// receives. Mirrors the ExtensionTarget enum in venom.allium.
type extensionTarget int

const (
	extTargetCobra extensionTarget = iota
	extTargetViper
)

// extension holds a developer-supplied callback registered via WithCobra or
// WithViper. position is a 0-based index into the App's extension list,
// satisfying invariants ExtensionPositionsAreUnique and
// ExtensionPositionsAreValid in venom.allium.
type extension struct {
	position int
	target   extensionTarget
	cobra    func(*cobra.Command) error
	viper    func(*viper.Viper) error
}

// WithCobra registers a callback that receives the constructed cobra root
// command after Venom builds the command tree. The callback may freely
// mutate the root: add subcommands, override hooks, customise help, etc.
//
// Multiple WithCobra/WithViper extensions are invoked in declaration order.
// Returning a non-nil error aborts preparation: during Execute the error is
// printed to stderr and the process exits 1; during Build the error is
// returned to the caller and no artifacts are produced.
func WithCobra(fn func(*cobra.Command) error) Option {
	return func(cfg *appConfig) {
		cfg.extensions = append(cfg.extensions, extension{
			position: len(cfg.extensions),
			target:   extTargetCobra,
			cobra:    fn,
		})
	}
}

// WithViper registers a callback that receives the per-app viper instance
// after Venom configures it (env prefix, config paths, config file read).
// The callback may freely mutate the instance: add config paths, re-read
// config, set overrides, etc.
//
// Same ordering and error semantics as WithCobra.
func WithViper(fn func(*viper.Viper) error) Option {
	return func(cfg *appConfig) {
		cfg.extensions = append(cfg.extensions, extension{
			position: len(cfg.extensions),
			target:   extTargetViper,
			viper:    fn,
		})
	}
}

// applyExtensions implements rule ApplyExtensions in venom.allium: invokes
// each registered extension in declaration order, passing the matching
// artifact. Returns the first non-nil error from a callback; subsequent
// extensions are not invoked.
func applyExtensions(extensions []extension, root *cobra.Command, v *viper.Viper) error {
	for _, ext := range extensions {
		switch ext.target {
		case extTargetCobra:
			if ext.cobra != nil {
				if err := ext.cobra(root); err != nil {
					return err
				}
			}
		case extTargetViper:
			if ext.viper != nil {
				if err := ext.viper(v); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
