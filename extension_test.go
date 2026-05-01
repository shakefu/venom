package venom

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestWithCobraRegistersExtension verifies that WithCobra adds an Extension
// of target=cobra to the App's extension list.
func TestWithCobraRegistersExtension(t *testing.T) {
	called := false
	app := New(
		WithName("testapp"),
		WithCobra(func(root *cobra.Command) error {
			called = true
			return nil
		}),
	)

	if got := len(app.cfg.extensions); got != 1 {
		t.Fatalf("expected 1 extension after WithCobra, got %d", got)
	}
	if app.cfg.extensions[0].target != extTargetCobra {
		t.Errorf("expected target=cobra, got %v", app.cfg.extensions[0].target)
	}

	// Sanity: callback runs when Build is invoked.
	if _, _, err := app.Build(); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !called {
		t.Error("cobra extension callback was never invoked")
	}
}

// TestWithViperRegistersExtension verifies that WithViper adds an Extension
// of target=viper to the App's extension list.
func TestWithViperRegistersExtension(t *testing.T) {
	called := false
	app := New(
		WithName("testapp"),
		WithViper(func(v *viper.Viper) error {
			called = true
			return nil
		}),
	)

	if got := len(app.cfg.extensions); got != 1 {
		t.Fatalf("expected 1 extension after WithViper, got %d", got)
	}
	if app.cfg.extensions[0].target != extTargetViper {
		t.Errorf("expected target=viper, got %v", app.cfg.extensions[0].target)
	}

	if _, _, err := app.Build(); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !called {
		t.Error("viper extension callback was never invoked")
	}
}

// TestExtensionsFireInDeclarationOrder verifies that ApplyExtensions invokes
// callbacks in the order they were declared on the App, regardless of target.
func TestExtensionsFireInDeclarationOrder(t *testing.T) {
	var calls []string
	app := New(
		WithName("testapp"),
		WithCobra(func(*cobra.Command) error { calls = append(calls, "cobra1"); return nil }),
		WithViper(func(*viper.Viper) error { calls = append(calls, "viper1"); return nil }),
		WithCobra(func(*cobra.Command) error { calls = append(calls, "cobra2"); return nil }),
		WithViper(func(*viper.Viper) error { calls = append(calls, "viper2"); return nil }),
	)

	if _, _, err := app.Build(); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	want := []string{"cobra1", "viper1", "cobra2", "viper2"}
	if len(calls) != len(want) {
		t.Fatalf("expected %d calls, got %d: %v", len(want), len(calls), calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Errorf("call[%d] = %q, want %q (full order: %v)", i, calls[i], want[i], calls)
		}
	}
}

// TestExtensionPositionsAreUniqueAndValid verifies invariants
// ExtensionPositionsAreUnique and ExtensionPositionsAreValid: positions are
// 0-based, dense, and unique across an App's extension list.
func TestExtensionPositionsAreUniqueAndValid(t *testing.T) {
	app := New(
		WithName("testapp"),
		WithCobra(func(*cobra.Command) error { return nil }),
		WithViper(func(*viper.Viper) error { return nil }),
		WithCobra(func(*cobra.Command) error { return nil }),
	)

	exts := app.cfg.extensions
	if len(exts) != 3 {
		t.Fatalf("expected 3 extensions, got %d", len(exts))
	}

	seen := make(map[int]bool, len(exts))
	for _, e := range exts {
		if e.position < 0 || e.position >= len(exts) {
			t.Errorf("extension position %d outside [0, %d)", e.position, len(exts))
		}
		if seen[e.position] {
			t.Errorf("duplicate extension position %d", e.position)
		}
		seen[e.position] = true
	}

	for i := 0; i < len(exts); i++ {
		if !seen[i] {
			t.Errorf("missing position %d (positions are not dense)", i)
		}
	}
}

// TestCobraExtensionReceivesRoot verifies that a cobra extension receives the
// constructed root cobra.Command (with the app's name as Use).
func TestCobraExtensionReceivesRoot(t *testing.T) {
	var got *cobra.Command
	app := New(
		WithName("rooty"),
		WithCobra(func(root *cobra.Command) error { got = root; return nil }),
	)

	if _, _, err := app.Build(); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if got == nil {
		t.Fatal("cobra extension never received a *cobra.Command")
	}
	if got.Use != "rooty" {
		t.Errorf("expected root.Use=%q, got %q", "rooty", got.Use)
	}
}

// TestViperExtensionReceivesInstance verifies that a viper extension receives
// a non-nil *viper.Viper.
func TestViperExtensionReceivesInstance(t *testing.T) {
	var got *viper.Viper
	app := New(
		WithName("vipy"),
		WithViper(func(v *viper.Viper) error { got = v; return nil }),
	)

	if _, _, err := app.Build(); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if got == nil {
		t.Fatal("viper extension never received a *viper.Viper")
	}
}

// TestExtensionFailureAbortsExecute verifies rule ExtensionFailureAbortsExecute:
// a non-nil error from an extension during Execute aborts the run; the
// remaining extensions and fang.Execute are skipped.
//
// We test this through the unexported run() method (which Execute wraps with
// os.Exit). run() returns the extension's error directly.
func TestExtensionFailureAbortsExecute(t *testing.T) {
	wantErr := errors.New("boom")
	secondCalled := false

	app := New(
		WithName("failapp"),
		WithCobra(func(*cobra.Command) error { return wantErr }),
		WithCobra(func(*cobra.Command) error { secondCalled = true; return nil }),
	)

	err := app.run(context.Background(), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected run() to return %v, got %v", wantErr, err)
	}
	if secondCalled {
		t.Error("second extension was invoked after first failed; preparation should abort on first error")
	}
}

// TestExtensionFailureAbortsBuild verifies rule ExtensionFailureAbortsBuild:
// a non-nil error from an extension during Build returns the error to the
// caller and produces no artifacts (cobra root and viper instance are nil).
func TestExtensionFailureAbortsBuild(t *testing.T) {
	wantErr := errors.New("kaboom")

	app := New(
		WithName("failapp"),
		WithViper(func(*viper.Viper) error { return wantErr }),
	)

	root, v, err := app.Build()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected Build() to return %v, got %v", wantErr, err)
	}
	if root != nil {
		t.Error("expected nil cobra root on failed Build, got non-nil")
	}
	if v != nil {
		t.Error("expected nil viper instance on failed Build, got non-nil")
	}
}
