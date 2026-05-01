package venom

import (
	"context"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// TestPositionalArgRuntimeResolution verifies rules ResolvePositionalArgValue
// and ValidateRequiredPositionalArgs in venom.allium. It exercises all four
// resolution paths inside makeRunFunc:
//   - variadic with args present  → args[position:]
//   - variadic with no args       → empty []string
//   - non-variadic with arg       → convertArg(args[position], type)
//   - non-variadic absent + default → convertArg(default, type)
//   - non-variadic absent + zero  → zeroForType(type)
//   - required arg missing        → error
//
// The function under test takes (src required, dst optional with default,
// more variadic), so each subtest can assert the values the function
// received from the runtime resolution.
func TestPositionalArgRuntimeResolution(t *testing.T) {
	var captured struct {
		src  string
		dst  string
		more []string
	}
	fn := func(ctx context.Context, src string, dst string, more []string) error {
		captured.src = src
		captured.dst = dst
		captured.more = more
		return nil
	}

	meta := &FuncMeta{
		FullName:    "test.posargs",
		CommandPath: []string{"posargs"},
		PositionalArgs: []PositionalArgMeta{
			{Name: "src", Type: "string", Position: 0, Cardinality: ArgRequired},
			{Name: "dst", Type: "string", Position: 1, Cardinality: ArgOptional, Default: "default-dst"},
			{Name: "more", Type: "[]string", Position: 2, Cardinality: ArgVariadic},
		},
		Func: fn,
	}

	tests := []struct {
		name     string
		args     []string
		wantSrc  string
		wantDst  string
		wantMore []string
		wantErr  bool
	}{
		{
			name:     "required_only_uses_optional_default_and_empty_variadic",
			args:     []string{"a"},
			wantSrc:  "a",
			wantDst:  "default-dst",
			wantMore: []string{},
		},
		{
			name:     "required_and_optional_present",
			args:     []string{"a", "b"},
			wantSrc:  "a",
			wantDst:  "b",
			wantMore: []string{},
		},
		{
			name:     "variadic_collects_remaining",
			args:     []string{"a", "b", "c", "d"},
			wantSrc:  "a",
			wantDst:  "b",
			wantMore: []string{"c", "d"},
		},
		{
			name:    "required_missing_returns_error",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured.src = ""
			captured.dst = ""
			captured.more = nil

			v := viper.New()
			cmd := buildCommand(meta, v)
			cmd.SetContext(context.Background())

			err := cmd.RunE(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunE err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if captured.src != tt.wantSrc {
				t.Errorf("src = %q, want %q", captured.src, tt.wantSrc)
			}
			if captured.dst != tt.wantDst {
				t.Errorf("dst = %q, want %q", captured.dst, tt.wantDst)
			}
			if !reflect.DeepEqual(captured.more, tt.wantMore) {
				t.Errorf("more = %#v, want %#v", captured.more, tt.wantMore)
			}
		})
	}
}

// TestPositionalArgZeroFallback verifies that an optional positional arg
// with no @default falls back to the type's zero value (per
// ResolvePositionalArgValue's first_of chain).
func TestPositionalArgZeroFallback(t *testing.T) {
	var captured struct {
		count int
		flag  bool
	}
	fn := func(ctx context.Context, count int, flag bool) error {
		captured.count = count
		captured.flag = flag
		return nil
	}

	meta := &FuncMeta{
		FullName:    "test.zerofallback",
		CommandPath: []string{"zerofallback"},
		PositionalArgs: []PositionalArgMeta{
			{Name: "count", Type: "int", Position: 0, Cardinality: ArgOptional},
			{Name: "flag", Type: "bool", Position: 1, Cardinality: ArgOptional},
		},
		Func: fn,
	}

	v := viper.New()
	cmd := buildCommand(meta, v)
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("RunE err = %v", err)
	}
	if captured.count != 0 {
		t.Errorf("count = %d, want 0 (int zero value)", captured.count)
	}
	if captured.flag != false {
		t.Errorf("flag = %v, want false (bool zero value)", captured.flag)
	}
}

// TestConvertArg verifies the helper used by makeRunFunc to convert string
// CLI arguments to typed values.
func TestConvertArg(t *testing.T) {
	tests := []struct {
		name string
		s    string
		typ  string
		want interface{}
	}{
		{"string_passthrough", "hello", "string", "hello"},
		{"int", "42", "int", 42},
		{"int64", "9000000000", "int64", int64(9000000000)},
		{"float64", "3.14", "float64", 3.14},
		{"bool_true", "true", "bool", true},
		{"bool_false", "false", "bool", false},
		{"unknown_type_falls_through_to_string", "x", "?", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertArg(tt.s, tt.typ)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertArg(%q, %q) = %v (%T), want %v (%T)",
					tt.s, tt.typ, got, got, tt.want, tt.want)
			}
		})
	}
}

// TestZeroForType verifies the zero values returned for each supported
// positional arg type.
func TestZeroForType(t *testing.T) {
	tests := []struct {
		typ  string
		want interface{}
	}{
		{"string", ""},
		{"int", 0},
		{"int64", int64(0)},
		{"float64", float64(0)},
		{"bool", false},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			got := zeroForType(tt.typ)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("zeroForType(%q) = %v (%T), want %v (%T)",
					tt.typ, got, got, tt.want, tt.want)
			}
		})
	}
}
