package main

import (
	"fmt"
	"os"

	"github.com/shakefu/venom/gen"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "venom",
		Short: "Venom CLI code generator",
	}

	generate := &cobra.Command{
		Use:   "generate [dir]",
		Short: "Generate venom registration code for a package",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return gen.Generate(dir)
		},
	}

	root.AddCommand(generate)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
