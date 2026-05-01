package venom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// TestDefaultConfigPathsAreSearched verifies the spec config
// `default_config_paths: Set<String> = {".", "$HOME"}` in venom.allium:120-122
// is honoured: when an appConfig has no explicit configPaths, setupViper
// searches the current directory.
//
// We assert this by writing a config file into a temp dir, chdir'ing into
// it (so "." resolves to the temp dir), and verifying setupViper loads
// the file's contents into the supplied viper instance.
func TestDefaultConfigPathsAreSearched(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".testapp.yaml")
	if err := os.WriteFile(configPath, []byte("foo: bar\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg := &appConfig{appName: "testapp"} // no configPaths → use defaults
	v := viper.New()
	if err := setupViper(cfg, v); err != nil {
		t.Fatalf("setupViper: %v", err)
	}

	if got := v.GetString("foo"); got != "bar" {
		t.Errorf(`expected setupViper to load .testapp.yaml from "." in default paths; foo = %q, want "bar"`, got)
	}
}
