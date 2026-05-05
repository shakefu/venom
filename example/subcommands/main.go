// Subcommands example: function names with underscores produce a command
// hierarchy. `serve_http` becomes `serve http`, `db_migrate` becomes
// `db migrate`, and so on. CamelCase segments are kebab-cased
// independently — `db_seedDev` becomes `db seed-dev`.
package main

import (
	"context"
	"fmt"

	"github.com/shakefu/venom"
)

//go:generate venom generate

// @cmd serve over HTTP
func serve_http(
	ctx context.Context,
	port int, // @short p @default 8080 @desc "port to listen on"
) error {
	fmt.Printf("HTTP server on :%d\n", port)
	return nil
}

// @cmd serve over gRPC
func serve_grpc(
	ctx context.Context,
	port int, // @short p @default 9090 @desc "port to listen on"
) error {
	fmt.Printf("gRPC server on :%d\n", port)
	return nil
}

// @cmd apply pending migrations
func db_migrate(
	ctx context.Context,
	dryRun bool, // @desc "print actions without applying them"
) error {
	if dryRun {
		fmt.Println("would migrate")
		return nil
	}
	fmt.Println("migrating")
	return nil
}

// @cmd seed development data
func db_seedDev(ctx context.Context) error {
	fmt.Println("seeding dev data")
	return nil
}

// @cmd reset the database
func db_reset(
	ctx context.Context,
	confirm bool, // @required @desc "must be true to reset"
) error {
	if !confirm {
		fmt.Println("refusing to reset without --confirm")
		return nil
	}
	fmt.Println("resetting")
	return nil
}

func main() {
	venom.Execute(serve_http, serve_grpc, db_migrate, db_seedDev, db_reset)
}
