package main

import (
	"context"
	"fmt"

	"github.com/shakefu/venom"
)

// @cmd starts the HTTP server
func serve(
	ctx context.Context,
	port int, // @short p @default 8080 @desc "port to listen on"
	host string, // @default localhost @desc "host to bind"
) error {
	fmt.Printf("Listening on %s:%d\n", host, port)
	return nil
}

// @cmd initialize a new project
func initProject(
	ctx context.Context,
	dir string, // @default . @desc "directory to initialize"
) error {
	fmt.Printf("Initializing project in %s\n", dir)
	return nil
}

// @cmd show version information
func version(ctx context.Context) error {
	fmt.Println("example v0.1.0")
	return nil
}

// @cmd copy files to destination
func copyFiles(
	ctx context.Context,
	src string, // @arg @required @desc "source path"
	dst string, // @arg @desc "destination path"
	extra []string, // @arg @desc "additional files"
	verbose bool, // @short v @desc "enable verbose output"
) error {
	return nil
}

func main() {
	venom.Execute(serve, initProject, version, copyFiles)
}
