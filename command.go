package venom

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// buildCommandTree creates a root cobra.Command and attaches a sub-command for
// each FuncMeta at the appropriate depth in the tree.
func buildCommandTree(appName string, metas []*FuncMeta) *cobra.Command {
	root := &cobra.Command{
		Use: appName,
	}

	for _, meta := range metas {
		cmd := buildCommand(meta)
		attachCommand(root, meta.CommandPath, cmd)
	}

	return root
}

// buildCommand creates a single cobra.Command from a FuncMeta.
func buildCommand(meta *FuncMeta) *cobra.Command {
	use := meta.CommandPath[len(meta.CommandPath)-1]
	for _, pa := range meta.PositionalArgs {
		switch pa.Cardinality {
		case ArgRequired:
			use += " <" + pa.Name + ">"
		case ArgOptional:
			use += " [" + pa.Name + "]"
		case ArgVariadic:
			use += " [" + pa.Name + "...]"
		}
	}

	cmd := &cobra.Command{
		Use:   use,
		Short: meta.Description,
		RunE:  makeRunFunc(meta),
	}

	if len(meta.PositionalArgs) > 0 {
		cmd.Args = makeArgsValidator(meta.PositionalArgs)
	}

	for _, p := range meta.Params {
		registerFlag(cmd, p)
	}

	return cmd
}

// makeArgsValidator returns a cobra positional arg validator based on the
// declared positional args metadata.
func makeArgsValidator(posArgs []PositionalArgMeta) cobra.PositionalArgs {
	var requiredCount int
	var hasVariadic bool
	totalNonVariadic := 0

	for _, pa := range posArgs {
		switch pa.Cardinality {
		case ArgRequired:
			requiredCount++
			totalNonVariadic++
		case ArgOptional:
			totalNonVariadic++
		case ArgVariadic:
			hasVariadic = true
		}
	}

	if hasVariadic {
		return cobra.MinimumNArgs(requiredCount)
	}
	if totalNonVariadic > requiredCount {
		return cobra.RangeArgs(requiredCount, totalNonVariadic)
	}
	return cobra.ExactArgs(requiredCount)
}

// attachCommand walks (or creates) intermediate commands so that cmd is placed
// at the correct depth. For path ["init", "project"], an "init" parent is
// created if it does not already exist, and "project" is added as its child.
func attachCommand(root *cobra.Command, path []string, cmd *cobra.Command) {
	parent := root

	// Walk all segments except the last one, creating placeholders as needed.
	for _, seg := range path[:len(path)-1] {
		var found *cobra.Command
		for _, child := range parent.Commands() {
			if child.Name() == seg {
				found = child
				break
			}
		}
		if found == nil {
			found = &cobra.Command{
				Use: seg,
			}
			parent.AddCommand(found)
		}
		parent = found
	}

	parent.AddCommand(cmd)
}

// registerFlag adds a flag to cmd based on the parameter's type.
func registerFlag(cmd *cobra.Command, p ParamMeta) {
	flags := cmd.Flags()

	switch p.Type {
	case "string":
		def := p.Default
		if p.Short != "" {
			flags.StringP(p.FlagName, p.Short, def, p.Desc)
		} else {
			flags.String(p.FlagName, def, p.Desc)
		}

	case "int":
		def, _ := strconv.Atoi(p.Default)
		if p.Short != "" {
			flags.IntP(p.FlagName, p.Short, def, p.Desc)
		} else {
			flags.Int(p.FlagName, def, p.Desc)
		}

	case "int64":
		def, _ := strconv.ParseInt(p.Default, 10, 64)
		if p.Short != "" {
			flags.Int64P(p.FlagName, p.Short, def, p.Desc)
		} else {
			flags.Int64(p.FlagName, def, p.Desc)
		}

	case "float64":
		def, _ := strconv.ParseFloat(p.Default, 64)
		if p.Short != "" {
			flags.Float64P(p.FlagName, p.Short, def, p.Desc)
		} else {
			flags.Float64(p.FlagName, def, p.Desc)
		}

	case "bool":
		def := p.Default == "true"
		if p.Short != "" {
			flags.BoolP(p.FlagName, p.Short, def, p.Desc)
		} else {
			flags.Bool(p.FlagName, def, p.Desc)
		}

	case "[]string":
		var def []string
		if p.Short != "" {
			flags.StringSliceP(p.FlagName, p.Short, def, p.Desc)
		} else {
			flags.StringSlice(p.FlagName, def, p.Desc)
		}

	case "time.Duration":
		var def time.Duration
		if p.Default != "" {
			def, _ = time.ParseDuration(p.Default)
		}
		if p.Short != "" {
			flags.DurationP(p.FlagName, p.Short, def, p.Desc)
		} else {
			flags.Duration(p.FlagName, def, p.Desc)
		}
	}
}

// makeRunFunc returns a RunE closure that reads flag values from viper and
// invokes the underlying function via reflection.
func makeRunFunc(meta *FuncMeta) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Build argument list: first arg is always context.Context.
		fnArgs := make([]reflect.Value, 0, 1+len(meta.Params)+len(meta.PositionalArgs))
		fnArgs = append(fnArgs, reflect.ValueOf(ctx))

		for _, p := range meta.Params {
			key := flagToConfigKey(p.FlagName)
			var val interface{}

			switch p.Type {
			case "string":
				val = viper.GetString(key)
			case "int":
				val = viper.GetInt(key)
			case "int64":
				val = viper.GetInt64(key)
			case "float64":
				val = viper.GetFloat64(key)
			case "bool":
				val = viper.GetBool(key)
			case "[]string":
				val = viper.GetStringSlice(key)
			case "time.Duration":
				val = viper.GetDuration(key)
			default:
				return fmt.Errorf("unsupported parameter type %q for flag %q", p.Type, p.FlagName)
			}

			// Validate required flags. At this point viper has merged CLI
			// flags, environment variables, and config file values, so any
			// source satisfies the requirement.
			if p.Required && !cmd.Flags().Lookup(p.FlagName).Changed {
				if isZero(val) {
					return fmt.Errorf("required flag %q not set", p.FlagName)
				}
			}

			fnArgs = append(fnArgs, reflect.ValueOf(val))
		}

		// Validate and collect positional arguments.
		for _, pa := range meta.PositionalArgs {
			if pa.Cardinality == ArgRequired && pa.Position >= len(args) {
				return fmt.Errorf("required argument %q not provided", pa.Name)
			}
		}

		for _, pa := range meta.PositionalArgs {
			var val interface{}
			switch {
			case pa.Cardinality == ArgVariadic:
				if pa.Position < len(args) {
					val = args[pa.Position:]
				} else {
					val = []string{}
				}
			default:
				if pa.Position < len(args) {
					val = convertArg(args[pa.Position], pa.Type)
				} else if pa.Default != "" {
					val = convertArg(pa.Default, pa.Type)
				} else {
					val = zeroForType(pa.Type)
				}
			}
			fnArgs = append(fnArgs, reflect.ValueOf(val))
		}

		results := reflect.ValueOf(meta.Func).Call(fnArgs)

		// The function is expected to return a single error value.
		if len(results) > 0 {
			last := results[len(results)-1]
			if !last.IsNil() {
				return last.Interface().(error)
			}
		}

		return nil
	}
}

// isZero reports whether val is the zero value for its type.
func isZero(val interface{}) bool {
	if val == nil {
		return true
	}
	return reflect.DeepEqual(val, reflect.Zero(reflect.TypeOf(val)).Interface())
}

// convertArg converts a string argument to the given Go type.
func convertArg(s string, typ string) interface{} {
	switch typ {
	case "int":
		v, _ := strconv.Atoi(s)
		return v
	case "int64":
		v, _ := strconv.ParseInt(s, 10, 64)
		return v
	case "float64":
		v, _ := strconv.ParseFloat(s, 64)
		return v
	case "bool":
		v, _ := strconv.ParseBool(s)
		return v
	case "time.Duration":
		v, _ := time.ParseDuration(s)
		return v
	default: // "string" and anything else
		return s
	}
}

// zeroForType returns the zero value for the given Go type string.
func zeroForType(typ string) interface{} {
	switch typ {
	case "int":
		return 0
	case "int64":
		return int64(0)
	case "float64":
		return float64(0)
	case "bool":
		return false
	case "time.Duration":
		return time.Duration(0)
	default: // "string"
		return ""
	}
}
