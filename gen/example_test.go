package gen_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shakefu/venom/gen"
)

func ExampleGenerate() {
	dir, err := os.MkdirTemp("", "venom-gen-*")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer os.RemoveAll(dir)

	src := `package main

import (
	"context"
	"fmt"

	"github.com/shakefu/venom"
)

// @cmd greet the user
func hello(ctx context.Context, name string) error {
	fmt.Printf("Hello, %s!\n", name)
	return nil
}

func main() {
	venom.Execute(hello)
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0644); err != nil {
		fmt.Println("error:", err)
		return
	}

	if err := gen.Generate(dir); err != nil {
		fmt.Println("error:", err)
		return
	}

	out, err := os.ReadFile(filepath.Join(dir, "venom_gen.go"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	content := string(out)
	fmt.Println(strings.Contains(content, `"main.hello"`))
	fmt.Println(strings.Contains(content, `"greet the user"`))
	// Output:
	// true
	// true
}
