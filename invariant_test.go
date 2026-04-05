package venom

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// dummyFunc is a placeholder for FuncMeta.Func so that buildCommand can create
// a cobra.Command without nil-pointer issues in RunE reflection.
func dummyFunc(_ context.Context) error { return nil }

// TestUniqueShortFlags verifies that within a single command, no two params may
// share the same short flag letter. Cobra panics when duplicate shorthand flags
// are registered on the same FlagSet, so we assert that the panic occurs.
func TestUniqueShortFlags(t *testing.T) {
	tests := []struct {
		name        string
		params      []ParamMeta
		shouldPanic bool
	}{
		{
			name: "distinct short flags are accepted",
			params: []ParamMeta{
				{Name: "host", Type: "string", FlagName: "host", Short: "H", Desc: "hostname"},
				{Name: "port", Type: "int", FlagName: "port", Short: "p", Desc: "port number"},
			},
			shouldPanic: false,
		},
		{
			name: "duplicate short flags panic",
			params: []ParamMeta{
				{Name: "host", Type: "string", FlagName: "host", Short: "p", Desc: "hostname"},
				{Name: "port", Type: "int", FlagName: "port", Short: "p", Desc: "port number"},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &FuncMeta{
				FullName:    "main.test",
				CommandPath: []string{"test"},
				Description: "test command",
				Params:      tt.params,
				Func:        dummyFunc,
			}

			panicked := didPanic(func() {
				buildCommand(meta)
			})

			if tt.shouldPanic && !panicked {
				t.Error("expected panic for duplicate short flags, but none occurred")
			}
			if !tt.shouldPanic && panicked {
				t.Error("unexpected panic for unique short flags")
			}
		})
	}
}

// TestUniqueFlagNames verifies that within a single command, no two params may
// share the same flag name. Cobra panics on duplicate flag names.
func TestUniqueFlagNames(t *testing.T) {
	tests := []struct {
		name        string
		params      []ParamMeta
		shouldPanic bool
	}{
		{
			name: "distinct flag names are accepted",
			params: []ParamMeta{
				{Name: "host", Type: "string", FlagName: "host", Desc: "hostname"},
				{Name: "port", Type: "int", FlagName: "port", Desc: "port number"},
			},
			shouldPanic: false,
		},
		{
			name: "duplicate flag names panic",
			params: []ParamMeta{
				{Name: "host", Type: "string", FlagName: "addr", Desc: "hostname"},
				{Name: "port", Type: "int", FlagName: "addr", Desc: "port number"},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &FuncMeta{
				FullName:    "main.test",
				CommandPath: []string{"test"},
				Description: "test command",
				Params:      tt.params,
				Func:        dummyFunc,
			}

			panicked := didPanic(func() {
				buildCommand(meta)
			})

			if tt.shouldPanic && !panicked {
				t.Error("expected panic for duplicate flag names, but none occurred")
			}
			if !tt.shouldPanic && panicked {
				t.Error("unexpected panic for unique flag names")
			}
		})
	}
}

// TestContextNeverAFlag verifies that context.Context parameters never appear as
// flags. By spec, codegen strips context.Context from Params, so FuncMeta.Params
// should never contain a context type. This test builds a command from a FuncMeta
// with typical params and asserts no flag is registered for "context.Context".
// It also verifies that if someone mistakenly included a context.Context param,
// it would not be registered as a flag (registerFlag ignores unknown types).
func TestContextNeverAFlag(t *testing.T) {
	tests := []struct {
		name   string
		params []ParamMeta
	}{
		{
			name: "normal params produce no context flag",
			params: []ParamMeta{
				{Name: "host", Type: "string", FlagName: "host", Desc: "hostname"},
				{Name: "port", Type: "int", FlagName: "port", Desc: "port number"},
			},
		},
		{
			name: "context.Context type param is ignored by registerFlag",
			params: []ParamMeta{
				{Name: "ctx", Type: "context.Context", FlagName: "ctx", Desc: "should not register"},
				{Name: "host", Type: "string", FlagName: "host", Desc: "hostname"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &FuncMeta{
				FullName:    "main.test",
				CommandPath: []string{"test"},
				Description: "test command",
				Params:      tt.params,
				Func:        dummyFunc,
			}

			cmd := buildCommand(meta)

			// No flag should have a context-related type.
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				// pflag stores type as a string, e.g. "string", "int", "bool".
				// A context.Context should never appear here.
				if strings.Contains(strings.ToLower(f.Name), "ctx") || strings.Contains(strings.ToLower(f.Value.Type()), "context") {
					t.Errorf("flag %q appears to be a context parameter (type %q); context.Context must never be a flag", f.Name, f.Value.Type())
				}
			})

			// Verify "ctx" is not a registered flag when a context.Context param
			// was included -- registerFlag should skip unknown types.
			if f := cmd.Flags().Lookup("ctx"); f != nil {
				t.Error("context.Context parameter was registered as a flag; it must be filtered out")
			}
		})
	}
}

// --- Positional-arg invariant tests ---

// TestRequiredBeforeOptional verifies spec obligation: required positional args
// must precede optional ones in the argument list.
func TestRequiredBeforeOptional(t *testing.T) {
	tests := []struct {
		name        string
		args        []PositionalArgMeta
		shouldPanic bool
	}{
		{
			name: "required_then_optional",
			args: []PositionalArgMeta{
				{Name: "src", Type: "string", Position: 0, Cardinality: ArgRequired},
				{Name: "dest", Type: "string", Position: 1, Cardinality: ArgOptional},
			},
			shouldPanic: false,
		},
		{
			name: "all_required",
			args: []PositionalArgMeta{
				{Name: "src", Type: "string", Position: 0, Cardinality: ArgRequired},
				{Name: "dest", Type: "string", Position: 1, Cardinality: ArgRequired},
			},
			shouldPanic: false,
		},
		{
			name: "optional_before_required",
			args: []PositionalArgMeta{
				{Name: "src", Type: "string", Position: 0, Cardinality: ArgOptional},
				{Name: "dest", Type: "string", Position: 1, Cardinality: ArgRequired},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicked := didPanic(func() {
				Register(&FuncMeta{
					FullName:       "test.required_before_optional_" + tt.name,
					CommandPath:    []string{"test"},
					PositionalArgs: tt.args,
				})
			})
			if tt.shouldPanic && !panicked {
				t.Error("expected panic")
			}
			if !tt.shouldPanic && panicked {
				t.Error("unexpected panic")
			}
		})
	}
}

// TestVariadicIsLast verifies spec obligation: a variadic positional arg must be
// the last positional arg.
func TestVariadicIsLast(t *testing.T) {
	tests := []struct {
		name        string
		args        []PositionalArgMeta
		shouldPanic bool
	}{
		{
			name: "variadic_last",
			args: []PositionalArgMeta{
				{Name: "src", Type: "string", Position: 0, Cardinality: ArgRequired},
				{Name: "files", Type: "[]string", Position: 1, Cardinality: ArgVariadic},
			},
			shouldPanic: false,
		},
		{
			name: "variadic_not_last",
			args: []PositionalArgMeta{
				{Name: "files", Type: "[]string", Position: 0, Cardinality: ArgVariadic},
				{Name: "dest", Type: "string", Position: 1, Cardinality: ArgRequired},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicked := didPanic(func() {
				Register(&FuncMeta{
					FullName:       "test.variadic_is_last_" + tt.name,
					CommandPath:    []string{"test"},
					PositionalArgs: tt.args,
				})
			})
			if tt.shouldPanic && !panicked {
				t.Error("expected panic")
			}
			if !tt.shouldPanic && panicked {
				t.Error("unexpected panic")
			}
		})
	}
}

// TestVariadicIsSlice verifies spec obligation: a variadic positional arg must
// have a slice type.
func TestVariadicIsSlice(t *testing.T) {
	tests := []struct {
		name        string
		args        []PositionalArgMeta
		shouldPanic bool
	}{
		{
			name: "variadic_slice_type",
			args: []PositionalArgMeta{
				{Name: "files", Type: "[]string", Position: 0, Cardinality: ArgVariadic},
			},
			shouldPanic: false,
		},
		{
			name: "variadic_non_slice_type",
			args: []PositionalArgMeta{
				{Name: "files", Type: "string", Position: 0, Cardinality: ArgVariadic},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicked := didPanic(func() {
				Register(&FuncMeta{
					FullName:       "test.variadic_is_slice_" + tt.name,
					CommandPath:    []string{"test"},
					PositionalArgs: tt.args,
				})
			})
			if tt.shouldPanic && !panicked {
				t.Error("expected panic")
			}
			if !tt.shouldPanic && panicked {
				t.Error("unexpected panic")
			}
		})
	}
}

// TestUniquePositionalPositions verifies spec obligation: no two positional args
// may share the same position index.
func TestUniquePositionalPositions(t *testing.T) {
	tests := []struct {
		name        string
		args        []PositionalArgMeta
		shouldPanic bool
	}{
		{
			name: "unique_positions",
			args: []PositionalArgMeta{
				{Name: "a", Type: "string", Position: 0, Cardinality: ArgRequired},
				{Name: "b", Type: "string", Position: 1, Cardinality: ArgRequired},
				{Name: "c", Type: "string", Position: 2, Cardinality: ArgRequired},
			},
			shouldPanic: false,
		},
		{
			name: "duplicate_positions",
			args: []PositionalArgMeta{
				{Name: "a", Type: "string", Position: 0, Cardinality: ArgRequired},
				{Name: "b", Type: "string", Position: 0, Cardinality: ArgRequired},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicked := didPanic(func() {
				Register(&FuncMeta{
					FullName:       "test.unique_positions_" + tt.name,
					CommandPath:    []string{"test"},
					PositionalArgs: tt.args,
				})
			})
			if tt.shouldPanic && !panicked {
				t.Error("expected panic")
			}
			if !tt.shouldPanic && panicked {
				t.Error("unexpected panic")
			}
		})
	}
}

// TestAtMostOneVariadic verifies spec obligation: at most one variadic positional
// arg is allowed per command.
func TestAtMostOneVariadic(t *testing.T) {
	tests := []struct {
		name        string
		args        []PositionalArgMeta
		shouldPanic bool
	}{
		{
			name: "one_variadic",
			args: []PositionalArgMeta{
				{Name: "files", Type: "[]string", Position: 0, Cardinality: ArgVariadic},
			},
			shouldPanic: false,
		},
		{
			name: "two_variadics",
			args: []PositionalArgMeta{
				{Name: "files", Type: "[]string", Position: 0, Cardinality: ArgVariadic},
				{Name: "dirs", Type: "[]string", Position: 1, Cardinality: ArgVariadic},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicked := didPanic(func() {
				Register(&FuncMeta{
					FullName:       "test.at_most_one_variadic_" + tt.name,
					CommandPath:    []string{"test"},
					PositionalArgs: tt.args,
				})
			})
			if tt.shouldPanic && !panicked {
				t.Error("expected panic")
			}
			if !tt.shouldPanic && panicked {
				t.Error("unexpected panic")
			}
		})
	}
}

// didPanic calls fn and reports whether it panicked.
func didPanic(fn func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	fn()
	return false
}
