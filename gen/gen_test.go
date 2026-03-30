package gen

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    map[string]string
	}{
		{
			name:    "short and default and desc",
			comment: `// @short p @default 8080 @desc "port to listen on"`,
			want:    map[string]string{"short": "p", "default": "8080", "desc": "port to listen on"},
		},
		{
			name:    "default and desc bare",
			comment: `// @default localhost @desc "host to bind"`,
			want:    map[string]string{"default": "localhost", "desc": "host to bind"},
		},
		{
			name:    "required flag",
			comment: `// @required @desc "must provide this"`,
			want:    map[string]string{"required": "true", "desc": "must provide this"},
		},
		{
			name:    "cmd annotation",
			comment: `// @cmd starts the HTTP server`,
			want:    map[string]string{"cmd": "starts the HTTP server"},
		},
		{
			name:    "empty comment",
			comment: `//`,
			want:    map[string]string{},
		},
		{
			name:    "no annotations",
			comment: `// just a regular comment`,
			want:    map[string]string{},
		},
		{
			name:    "desc with dot default",
			comment: `// @default . @desc "directory to initialize"`,
			want:    map[string]string{"default": ".", "desc": "directory to initialize"},
		},
		{
			name:    "only required",
			comment: `// @required`,
			want:    map[string]string{"required": "true"},
		},
		{
			name:    "short only",
			comment: `// @short v`,
			want:    map[string]string{"short": "v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAnnotations(tt.comment)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAnnotations(%q)\n  got  %v\n  want %v", tt.comment, got, tt.want)
			}
		})
	}
}

func TestFuncNameToCommandPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"simple", "serve", []string{"serve"}},
		{"camelCase", "initProject", []string{"init-project"}},
		{"underscore", "serve_tls", []string{"serve", "tls"}},
		{"camelCase segments", "init_fastProject", []string{"init", "fast-project"}},
		{"multi underscore", "init_project_fast", []string{"init", "project", "fast"}},
		{"package prefix", "main.serve", []string{"serve"}},
		{"package prefix with underscore", "main.init_project", []string{"init", "project"}},
		{"leading underscore", "_hidden", []string{"hidden"}},
		{"only underscores", "___", nil},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := funcNameToCommandPath(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcNameToCommandPath(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParamToFlagName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"port", "port"},
		{"serverPort", "server-port"},
		{"TLSConfig", "tls-config"},
		{"initProject", "init-project"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := paramToFlagName(tt.in)
			if got != tt.want {
				t.Errorf("paramToFlagName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	// Copy testdata/simple/input.go to a temp directory, run Generate, compare output.
	tmpDir := t.TempDir()

	input, err := os.ReadFile(filepath.Join("testdata", "simple", "input.go"))
	if err != nil {
		t.Fatalf("reading input: %v", err)
	}

	// Write the input file as main.go in the temp dir.
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), input, 0644); err != nil {
		t.Fatalf("writing main.go: %v", err)
	}

	// Run the generator.
	if err := Generate(tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Read the generated file.
	got, err := os.ReadFile(filepath.Join(tmpDir, "venom_gen.go"))
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	// Read the expected file.
	expected, err := os.ReadFile(filepath.Join("testdata", "simple", "expected_gen.go"))
	if err != nil {
		t.Fatalf("reading expected file: %v", err)
	}

	// Normalize line endings and trailing whitespace for comparison.
	gotStr := strings.TrimSpace(string(got))
	expectedStr := strings.TrimSpace(string(expected))

	if gotStr != expectedStr {
		t.Errorf("generated output does not match expected.\n--- GOT ---\n%s\n--- EXPECTED ---\n%s", gotStr, expectedStr)
	}
}
