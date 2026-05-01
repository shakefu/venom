package venom

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestBuildReturnsArtifacts verifies rule AppBuildSucceeds: when preparation
// completes without error, Build returns the constructed cobra root and viper
// instance with nil error.
func TestBuildReturnsArtifacts(t *testing.T) {
	app := New(WithName("buildy"))

	root, v, err := app.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if root == nil {
		t.Fatal("Build returned nil cobra root")
	}
	if v == nil {
		t.Fatal("Build returned nil viper instance")
	}
	if root.Use != "buildy" {
		t.Errorf("expected root.Use=%q, got %q", "buildy", root.Use)
	}
}

// TestBuildIsRepeatable verifies the App entity comment: "Each call produces
// a fresh cobra tree and viper instance; the App itself is reusable."
func TestBuildIsRepeatable(t *testing.T) {
	app := New(WithName("buildy"))

	root1, v1, err := app.Build()
	if err != nil {
		t.Fatalf("first Build failed: %v", err)
	}
	root2, v2, err := app.Build()
	if err != nil {
		t.Fatalf("second Build failed: %v", err)
	}

	if root1 == root2 {
		t.Error("expected fresh cobra root on second Build, got same pointer")
	}
	if v1 == v2 {
		t.Error("expected fresh viper instance on second Build, got same pointer")
	}
}

// TestBuildAppliesExtensions verifies that Build runs both cobra and viper
// extensions before returning, and that they receive non-nil artifacts.
func TestBuildAppliesExtensions(t *testing.T) {
	cobraSeen := false
	viperSeen := false

	app := New(
		WithName("buildy"),
		WithCobra(func(root *cobra.Command) error {
			if root == nil {
				t.Error("cobra extension received nil root")
			}
			cobraSeen = true
			return nil
		}),
		WithViper(func(v *viper.Viper) error {
			if v == nil {
				t.Error("viper extension received nil instance")
			}
			viperSeen = true
			return nil
		}),
	)

	if _, _, err := app.Build(); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !cobraSeen {
		t.Error("cobra extension was not invoked during Build")
	}
	if !viperSeen {
		t.Error("viper extension was not invoked during Build")
	}
}

// TestBuildExtensionMutationsArePersisted verifies that extensions can
// mutate their target and the mutations are reflected in the returned
// artifacts. This exercises the spec's "Callbacks may freely mutate their
// target" guarantee.
func TestBuildExtensionMutationsArePersisted(t *testing.T) {
	app := New(
		WithName("buildy"),
		WithCobra(func(root *cobra.Command) error {
			root.AddCommand(&cobra.Command{Use: "added-by-extension"})
			return nil
		}),
		WithViper(func(v *viper.Viper) error {
			v.Set("custom-key", "custom-value")
			return nil
		}),
	)

	root, v, err := app.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	var found bool
	for _, c := range root.Commands() {
		if c.Use == "added-by-extension" {
			found = true
			break
		}
	}
	if !found {
		t.Error("subcommand added by cobra extension is not present on returned root")
	}

	if got := v.GetString("custom-key"); got != "custom-value" {
		t.Errorf("viper extension mutation not visible: got %q, want %q", got, "custom-value")
	}
}
