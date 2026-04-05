# Venom

Declarative CLI generation for Go. Write functions, get commands.


```go
package main

import (
	"context"
	"fmt"

	"github.com/shakefu/venom"
)

//go:generate venom generate

// @cmd starts the HTTP server
func serve(
	ctx context.Context,
	port int,    // @short p @default 8080 @desc "port to listen on"
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
	fmt.Println("venom-example v0.1.0")
	return nil
}

func main() {
	venom.Execute(serve, initProject, version)
}
```

That's it. No structs, no builders, no boilerplate. Run `go generate` and you get:

![Venom styled help output](docs/venom-example.jpeg)

```
$ venom-example --help

  USAGE

    venom-example [command] [--flags]

  COMMANDS

    completion [command]    Generate the autocompletion script for the specified shell
    help [command]          Help about any command
    init-project [--flags]  Initialize a new project
    serve [--flags]         Starts the HTTP server
    version                 Show version information

  FLAGS

    -h --help               Help for venom-example
    -v --version            Version for venom-example
```

```
$ venom-example serve -p 3000 --host 0.0.0.0
Listening on 0.0.0.0:3000
```

## How it works

1. Write plain Go functions with annotations in comments
2. Add `//go:generate venom generate` to your package
3. Run `go generate` to produce registration code
4. Call `venom.Execute(...)` in main

Functions become commands. Parameters become flags. Parameters annotated with `@arg` become positional arguments. Annotations control the CLI behavior:

| Annotation | Where | Example |
|------------|-------|---------|
| `@cmd` | Function doc comment | `// @cmd starts the server` |
| `@desc` | Parameter comment | `// @desc "port to listen on"` |
| `@default` | Parameter comment | `// @default 8080` |
| `@short` | Parameter comment | `// @short p` |
| `@required` | Parameter comment | `// @required` |
| `@arg` | Parameter comment | `// @arg` |

<!-- AI-GENERATED -->
## Supported parameter types

| Go type | Flag example | Default parsing |
|---------|-------------|-----------------|
| `string` | `--host localhost` | Bare string |
| `int` | `--port 8080` | `strconv.Atoi` |
| `int64` | `--size 1024` | `strconv.ParseInt` |
| `float64` | `--rate 0.5` | `strconv.ParseFloat` |
| `bool` | `--verbose` | `"true"` / `"false"` |
| `[]string` | `--tags a,b,c` | Comma-separated |
| `time.Duration` | `--timeout 30s` | `time.ParseDuration` |

Unsupported types are silently ignored — the parameter will not appear as a flag.
Positional arguments (`@arg`) support the same types, with variadic args restricted to `[]string`.
<!-- /AI-GENERATED -->

## Naming conventions

Venom derives CLI names from Go names automatically:

| Go function | Command |
|-------------|---------|
| `serve` | `serve` |
| `initProject` | `init-project` |
| `serve_tls` | `serve tls` (subcommand) |

| Go parameter | Flag |
|--------------|------|
| `port` | `--port` |
| `serverPort` | `--server-port` |
| `useHTTPS` | `--use-https` |

## Configuration resolution

Flag values are resolved from multiple sources in priority order:

1. **CLI flag** — `--port 3000`
2. **Environment variable** — `VENOM_EXAMPLE_PORT=3000`
3. **Config file** — `.venom-example` (YAML, TOML, or JSON)
4. **Default** — `@default` annotation value
5. **Zero value** — type default (`0`, `""`, `false`)

> **Note:** Positional arguments (`@arg`) resolve from the command line only. They do not participate in environment variable, config file, or default resolution (except `@default` for optional args).

<!-- AI-GENERATED -->
Environment variables are formed from the app name prefix and the flag name in
SCREAMING_SNAKE_CASE. Hyphens in flag names become underscores:

| Flag | Environment variable |
|------|---------------------|
| `--port` | `VENOM_EXAMPLE_PORT` |
| `--server-port` | `VENOM_EXAMPLE_SERVER_PORT` |

Config files use the kebab-case flag name directly as the key. Venom
searches for a file named `.<appName>` (e.g. `.venom-example.yaml`) in `.` and
`$HOME` by default. Example `.venom-example.yaml`:

```yaml
port: 3000
host: 0.0.0.0
server-port: 9090
```
<!-- /AI-GENERATED -->

## Positional arguments

Parameters annotated with `@arg` become positional arguments instead of flags:

```go
// @cmd copy files to a destination
func copyFiles(
	ctx context.Context,
	src string,     // @arg @required @desc "source path"
	dst string,     // @arg @desc "destination path"
	extra []string, // @arg @desc "additional files"
	verbose bool,   // @short v @desc "enable verbose output"
) error {
	// src, dst, extra are positional; verbose is a flag
	return nil
}
```

```
$ myapp copy-files <src> [dst] [extra...]

$ myapp copy-files main.go /tmp
$ myapp copy-files main.go /tmp extra1.go extra2.go --verbose
```

Positional arguments:
- Are declared with `@arg` in the parameter comment
- Are ordered by their position in the function signature
- Support three cardinalities:
  - **Required** — `@arg @required` — must be provided
  - **Optional** — `@arg` — may be omitted (falls back to `@default` or zero value)
  - **Variadic** — `@arg` on a `[]string` parameter — collects all remaining arguments
- Do **not** participate in environment variable or config file resolution
- Can coexist with flags on the same command

Ordering rules:
- Required arguments must come before optional ones
- A variadic argument must be last
- At most one variadic argument per command

## Custom exit codes

Command functions return `error`. By default, errors exit with code 1. Implement `ErrorCode() int` for custom exit codes:

```go
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string   { return e.Message }
func (e *ValidationError) ErrorCode() int  { return 2 }
```

## Installation

```bash
go install github.com/shakefu/venom/cmd/venom@latest
```

Then in your project:

```bash
go get github.com/shakefu/venom
```

## Usage

Add a `go:generate` directive to your package and run `go generate`:

```go
//go:generate venom generate
```

```bash
go generate ./...
```

This produces a `venom_gen.go` file with `init()` registrations. Commit this file — it's part of your build.

### App configuration

For more control, use `venom.New()`:

```go
app := venom.New(
	venom.WithName("venom-example"),
	venom.WithVersion("1.0.0"),
	venom.WithEnvPrefix("VENOM_EXAMPLE"),
	venom.WithConfigName(".venom-example"),
	venom.WithConfigPaths(".", "$HOME"),
)
app.Execute(serve, initProject, version)
```

## Documentation

- [Allium specification](venom.allium) — formal domain specification

<!-- AI-GENERATED -->
## Troubleshooting

### `venom: no metadata registered for main.serve; did you run go generate?`

The function was passed to `venom.Execute()` but has no entry in `venom_gen.go`.
Run `go generate ./...` to regenerate registration code.

### `venom: expected a function, got <type>`

A non-function value was passed to `venom.Execute()`. Every argument must be a
function, not a method value or a non-function type.

### Panic: `venom: <func>: positional args "<a>" and "<b>" share position <N>`

Two `@arg` parameters have the same index in the function signature. Each
positional argument must occupy a unique position.

### Panic: `venom: <func>: multiple variadic positional args: [...]`

A command has more than one `[]string` parameter annotated with `@arg`. At most
one variadic argument is allowed per command.

### Panic: `venom: <func>: required positional arg "<name>" follows an optional arg`

Required `@arg @required` parameters must come before optional `@arg` parameters
in the function signature.

### Panic: `venom: <func>: variadic arg "<name>" must be after all other positional args`

The `[]string` variadic `@arg` parameter must be the last positional argument in
the function signature.

### Panic: `venom: <func>: variadic positional arg "<name>" must have type []string, got "<type>"`

Variadic positional arguments must be typed `[]string`. Other slice types are not
supported.

### A parameter does not appear as a flag

The parameter's Go type is not in the supported set (`string`, `int`, `int64`,
`float64`, `bool`, `[]string`, `time.Duration`). Unsupported types are silently
ignored. Use a supported type and parse inside the function body.

### Duplicate short flag panic at startup

Two parameters on the same command share the same `@short` letter. Cobra panics
when duplicate shorthand flags are registered. Use unique `@short` values or
remove the duplicate.
<!-- /AI-GENERATED -->

## Contributing

This project uses:

- [Conventional commits](https://www.conventionalcommits.org/) for commit messages
- [prek](https://github.com/j178/prek) for pre-commit hooks
- [cocogitto](https://docs.cocogitto.io/) for semantic releases

```bash
script/setup    # set up after cloning
script/test     # run tests
script/lint     # run linters
script/build    # build the project
```

## License

MIT
