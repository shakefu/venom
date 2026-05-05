// Build escape hatch example: when Execute is too opinionated, call Build to
// get the constructed cobra root and viper instance, then own the dispatch
// loop yourself.
//
// Build runs the same preparation pipeline as Execute (including any
// registered WithCobra/WithViper extensions) but returns the artifacts
// instead of dispatching. Compose them with hand-rolled commands, customise
// fang invocation, or skip fang entirely.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/shakefu/venom"
	"github.com/spf13/cobra"
)

//go:generate venom generate

// @cmd starts the HTTP server
func serve(
	ctx context.Context,
	port int, // @short p @default 8080 @desc "port to listen on"
) error {
	fmt.Printf("Listening on :%d\n", port)
	return nil
}

// @cmd initialize a new project
func initProject(
	ctx context.Context,
	dir string, // @default . @desc "directory to initialize"
) error {
	fmt.Printf("Initializing project in %s\n", dir)
	return nil
}

func main() {
	app := venom.New(
		venom.WithName("build-example"),
		venom.WithVersion("1.0.0"),
	)

	root, _, err := app.Build(serve, initProject)
	if err != nil {
		log.Fatal(err)
	}

	// Compose with hand-rolled subcommands after Build, before dispatch.
	root.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Run environment diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("doctor: all systems nominal")
			return nil
		},
	})

	// Dispatch yourself. This example uses cobra's native dispatch (no fang
	// styling). To keep fang's styled help, replace the line below with:
	//
	//	import "github.com/charmbracelet/fang"
	//	err := fang.Execute(ctx, root, fang.WithVersion("custom-1.0.0"))
	ctx := context.Background()
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
