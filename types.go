package venom

// ExitCoder is an optional interface that errors can implement to control the
// process exit code. If a command returns an error satisfying ExitCoder, its
// ErrorCode() value is used instead of the default 1.
type ExitCoder interface {
	ErrorCode() int
}

// ParamMeta describes a single function parameter extracted by codegen.
type ParamMeta struct {
	Name     string // Go parameter name, e.g. "port"
	Type     string // Go type as string, e.g. "int", "string", "[]string", "time.Duration"
	FlagName string // Derived kebab-case flag name
	Short    string // Single-char short flag (from @short)
	Default  string // Default value as string (from @default)
	Desc     string // Description (from @desc)
	Required bool   // Whether flag is required (from @required)
}

// ArgCardinality describes how a positional argument is consumed.
type ArgCardinality string

const (
	ArgRequired ArgCardinality = "required" // Exactly one value must be provided
	ArgOptional ArgCardinality = "optional" // Zero or one value may be provided
	ArgVariadic ArgCardinality = "variadic" // Zero or more values are collected
)

// PositionalArgMeta describes a single positional argument extracted by codegen.
type PositionalArgMeta struct {
	Name        string         // Go parameter name
	Type        string         // Go type as string
	Position    int            // 0-based index among positional args
	Cardinality ArgCardinality // required, optional, or variadic
	Default     string         // Default value as string (from @default)
	Desc        string         // Description (from @desc)
}

// FuncMeta describes a registered command function, populated by generated code.
type FuncMeta struct {
	FullName       string              // Runtime-qualified function name (e.g. "main.serve")
	CommandPath    []string            // Derived command hierarchy, e.g. ["serve"] or ["init", "project"]
	Description    string              // Command description (from @cmd)
	Params         []ParamMeta         // Parameter metadata (excluding context.Context)
	PositionalArgs []PositionalArgMeta // Positional argument metadata
	Func           interface{}         // Actual function value, set at runtime
}
