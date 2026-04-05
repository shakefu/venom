package venom_test

import (
	"fmt"

	"github.com/shakefu/venom"
)

// exitError demonstrates implementing venom.ExitCoder for custom exit codes.
type exitError struct {
	msg  string
	code int
}

func (e *exitError) Error() string  { return e.msg }
func (e *exitError) ErrorCode() int { return e.code }

func ExampleNew() {
	app := venom.New(
		venom.WithName("myapp"),
		venom.WithVersion("1.0.0"),
		venom.WithEnvPrefix("MYAPP"),
		venom.WithConfigName(".myapp"),
		venom.WithConfigPaths(".", "$HOME"),
	)
	fmt.Println(app != nil)
	// Output:
	// true
}

func ExampleRegister() {
	venom.Register(&venom.FuncMeta{
		FullName:    "example.serve",
		CommandPath: []string{"serve"},
		Description: "Start the HTTP server",
		Params: []venom.ParamMeta{
			{Name: "port", Type: "int", FlagName: "port", Short: "p", Default: "8080", Desc: "port to listen on"},
			{Name: "host", Type: "string", FlagName: "host", Default: "localhost", Desc: "host to bind"},
		},
	})
	fmt.Println("registered")
	// Output:
	// registered
}

func ExampleRegister_withPositionalArgs() {
	venom.Register(&venom.FuncMeta{
		FullName:    "example.copyFiles",
		CommandPath: []string{"copy-files"},
		Description: "Copy files to destination",
		Params: []venom.ParamMeta{
			{Name: "verbose", Type: "bool", FlagName: "verbose", Short: "v", Desc: "enable verbose output"},
		},
		PositionalArgs: []venom.PositionalArgMeta{
			{Name: "src", Type: "string", Position: 0, Cardinality: venom.ArgRequired, Desc: "source path"},
			{Name: "dst", Type: "string", Position: 1, Cardinality: venom.ArgOptional, Desc: "destination path"},
			{Name: "extra", Type: "[]string", Position: 2, Cardinality: venom.ArgVariadic, Desc: "additional files"},
		},
	})
	fmt.Println("registered")
	// Output:
	// registered
}

func ExampleExitCoder() {
	var err error = &exitError{msg: "validation failed", code: 2}

	if ec, ok := err.(venom.ExitCoder); ok {
		fmt.Println(ec.ErrorCode())
	}
	// Output:
	// 2
}
