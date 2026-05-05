// Extensions example: reach into the underlying cobra root and viper instance
// during App preparation without giving up Venom's codegen and resolution.
//
// Callbacks fire after Venom completes its internal setup (env prefix, config
// paths, config file read, command tree, persistent flag binding) and before
// the CLI dispatches.
package main

import (
	"context"
	"fmt"

	"github.com/shakefu/venom"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:generate venom generate

// @cmd starts the HTTP server
func serve(
	ctx context.Context,
	port int, // @short p @default 8080 @desc "port to listen on"
	host string, // @default localhost @desc "host to bind"
) error {
	fmt.Printf("Listening on %s:%d\n", host, port)
	return nil
}

func main() {
	app := venom.New(
		venom.WithName("ext-example"),
		venom.WithEnvPrefix("EXT_EXAMPLE"),

		// WithCobra runs after Venom builds the command tree. Use it to
		// silence usage on errors, register hand-rolled subcommands, attach
		// PersistentPreRun hooks, customise help, etc.
		venom.WithCobra(func(root *cobra.Command) error {
			root.SilenceUsage = true
			root.AddCommand(&cobra.Command{
				Use:   "ping",
				Short: "Hand-rolled subcommand added via WithCobra",
				RunE: func(cmd *cobra.Command, args []string) error {
					fmt.Println("pong")
					return nil
				},
			})
			return nil
		}),

		// WithViper runs after Venom configures viper (env prefix, config
		// paths, config file read). Use it to set defaults, add additional
		// config paths, register remote providers, etc.
		//
		// Note: v.Unmarshal(&cfg) uses `mapstructure` tags, not `yaml` or
		// `json`. See the README "Extension hooks" section.
		venom.WithViper(func(v *viper.Viper) error {
			v.AddConfigPath("/etc/ext-example")
			v.SetDefault("port", 9000)
			return nil
		}),
	)

	app.Execute(serve)
}
