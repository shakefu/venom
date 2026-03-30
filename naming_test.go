package venom

import (
	"reflect"
	"testing"
)

func TestFuncNameToCommandPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"simple", "serve", []string{"serve"}},
		{"camelCase", "initProject", []string{"init-project"}},
		{"underscore split", "serve_tls", []string{"serve", "tls"}},
		{"camelCase segments", "init_fastProject", []string{"init", "fast-project"}},
		{"package prefix", "main.serve", []string{"serve"}},
		{"package prefix with underscore", "main.init_project", []string{"init", "project"}},
		{"package prefix camelCase", "main.initProject", []string{"init-project"}},
		{"nested package", "pkg/sub.do_thing", []string{"do", "thing"}},
		{"leading underscore", "_hidden", []string{"hidden"}},
		{"trailing underscore", "serve_", []string{"serve"}},
		{"multiple underscores", "a__b", []string{"a", "b"}},
		{"only underscores", "___", nil},
		{"empty", "", nil},
		{"single char", "a", []string{"a"}},
		{"three segments", "init_project_fast", []string{"init", "project", "fast"}},
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
		name string
		in   string
		want string
	}{
		{"lowercase", "port", "port"},
		{"camelCase", "serverPort", "server-port"},
		{"PascalCase", "ServerPort", "server-port"},
		{"acronym start", "TLSConfig", "tls-config"},
		{"acronym middle", "useTLSConfig", "use-tls-config"},
		{"single char", "a", "a"},
		{"single upper", "A", "a"},
		{"all caps", "URL", "url"},
		{"empty", "", ""},
		{"already lower", "alreadylower", "alreadylower"},
		{"trailing acronym", "useHTTPS", "use-https"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := paramToFlagName(tt.in)
			if got != tt.want {
				t.Errorf("paramToFlagName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFlagToEnvVar(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		flag   string
		want   string
	}{
		{"basic", "MYAPP", "server-port", "MYAPP_SERVER_PORT"},
		{"no hyphens", "APP", "port", "APP_PORT"},
		{"empty prefix", "", "server-port", "SERVER_PORT"},
		{"single word", "X", "verbose", "X_VERBOSE"},
		{"multiple hyphens", "APP", "a-b-c", "APP_A_B_C"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flagToEnvVar(tt.prefix, tt.flag)
			if got != tt.want {
				t.Errorf("flagToEnvVar(%q, %q) = %q, want %q", tt.prefix, tt.flag, got, tt.want)
			}
		})
	}
}

func TestFlagToConfigKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"server-port", "server-port"},
		{"verbose", "verbose"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := flagToConfigKey(tt.in)
			if got != tt.want {
				t.Errorf("flagToConfigKey(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDeriveAppName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"full module path", "github.com/shakefu/venom", "venom"},
		{"two segments", "example.com/myapp", "myapp"},
		{"single segment", "myapp", "myapp"},
		{"empty", "", ""},
		{"trailing slash edge", "github.com/org/tool", "tool"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveAppName(tt.in)
			if got != tt.want {
				t.Errorf("deriveAppName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
