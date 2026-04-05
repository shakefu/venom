// Package gen implements code generation for the venom CLI framework.
//
// It parses Go source files to find venom.Execute() calls, extracts function
// metadata (parameter names, types, and comment annotations), and writes a
// venom_gen.go file containing init() code that registers each command.
package gen

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// funcInfo holds extracted metadata for a single command function.
type funcInfo struct {
	FuncName    string      // Original Go function name, e.g. "serve"
	FullName    string      // Runtime-qualified name, e.g. "main.serve"
	CommandPath []string    // Derived command hierarchy
	Description string      // From @cmd annotation on doc comment
	Params      []paramInfo // Extracted parameters (excluding context.Context)
	Args        []argInfo   // Positional arguments (from @arg annotation)
}

// paramInfo holds extracted metadata for a single function parameter.
type paramInfo struct {
	Name     string
	Type     string
	FlagName string
	Short    string
	Default  string
	Desc     string
	Required bool
}

// argInfo holds extracted metadata for a positional argument.
type argInfo struct {
	Name        string
	Type        string
	Position    int
	Cardinality string // "required", "optional", "variadic"
	Default     string
	Desc        string
}

// Generate is the main entry point. It parses all .go files in dir (excluding
// *_test.go and venom_gen.go), finds venom.Execute() calls, extracts function
// metadata, and writes venom_gen.go.
func Generate(dir string) error {
	fset := token.NewFileSet()

	// Parse all Go files in the directory.
	filter := func(fi os.FileInfo) bool {
		name := fi.Name()
		if strings.HasSuffix(name, "_test.go") {
			return false
		}
		if name == "venom_gen.go" {
			return false
		}
		return true
	}

	pkgs, err := parser.ParseDir(fset, dir, filter, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("gen: parsing directory %s: %w", dir, err)
	}

	for pkgName, pkg := range pkgs {
		files := make([]*ast.File, 0, len(pkg.Files))
		for _, f := range pkg.Files {
			files = append(files, f)
		}

		funcNames := findExecuteCalls(files)
		if len(funcNames) == 0 {
			continue
		}

		// Determine the package path for FullName.
		pkgPath := pkgName
		if pkgName != "main" {
			// For non-main packages, compute import path from go.mod.
			modPath, modDir := findModulePath(dir)
			if modPath != "" {
				rel, err := filepath.Rel(modDir, dir)
				if err == nil && rel != "." {
					pkgPath = modPath + "/" + filepath.ToSlash(rel)
				} else {
					pkgPath = modPath
				}
			}
		}

		var infos []*funcInfo
		for _, name := range funcNames {
			info, err := extractFuncMeta(fset, files, name, pkgPath)
			if err != nil {
				return err
			}
			infos = append(infos, info)
		}

		if err := writeGenFile(dir, pkgName, infos); err != nil {
			return err
		}
	}

	return nil
}

// parseAnnotations parses a comment string containing @-prefixed annotations.
//
// Supported annotations: @short, @default, @desc, @required, @cmd.
// Values can be bare words or quoted strings. @required has no value.
//
// Example: `@short p @default 8080 @desc "port to listen on"`
// Returns: {"short": "p", "default": "8080", "desc": "port to listen on"}
func parseAnnotations(comment string) map[string]string {
	result := make(map[string]string)

	// Strip leading // or /* */ comment markers.
	comment = strings.TrimSpace(comment)
	comment = strings.TrimPrefix(comment, "//")
	comment = strings.TrimPrefix(comment, "/*")
	comment = strings.TrimSuffix(comment, "*/")
	comment = strings.TrimSpace(comment)

	i := 0
	runes := []rune(comment)
	n := len(runes)

	for i < n {
		// Find the next @ tag.
		if runes[i] != '@' {
			i++
			continue
		}

		// Extract tag name.
		i++ // skip @
		tagStart := i
		for i < n && !unicode.IsSpace(runes[i]) && runes[i] != '@' {
			i++
		}
		tag := string(runes[tagStart:i])
		if tag == "" {
			continue
		}

		// Skip whitespace between tag and value.
		for i < n && unicode.IsSpace(runes[i]) {
			i++
		}

		// @required has no value.
		if tag == "required" {
			result[tag] = "true"
			continue
		}

		// If next char is @, the tag has no value.
		if i >= n || runes[i] == '@' {
			result[tag] = ""
			continue
		}

		// Extract value: quoted or bare word(s).
		var value string
		if runes[i] == '"' {
			// Quoted value: read until closing quote.
			i++ // skip opening quote
			valStart := i
			for i < n && runes[i] != '"' {
				i++
			}
			value = string(runes[valStart:i])
			if i < n {
				i++ // skip closing quote
			}
		} else {
			// Bare value: read until next @ or end.
			valStart := i
			for i < n && runes[i] != '@' {
				i++
			}
			value = strings.TrimSpace(string(runes[valStart:i]))
		}

		result[tag] = value
	}

	return result
}

// findExecuteCalls walks the AST to find venom.Execute(...) or <recv>.Execute(...)
// calls and returns the function name arguments.
func findExecuteCalls(files []*ast.File) []string {
	var funcNames []string

	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			if !isExecuteCall(call) {
				return true
			}

			// Extract function name identifiers from arguments.
			for _, arg := range call.Args {
				if ident, ok := arg.(*ast.Ident); ok {
					funcNames = append(funcNames, ident.Name)
				}
			}

			return true
		})
	}

	return funcNames
}

// isExecuteCall checks whether a call expression is a venom.Execute() or
// <receiver>.Execute() call.
func isExecuteCall(call *ast.CallExpr) bool {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		if fn.Sel.Name != "Execute" {
			return false
		}
		// venom.Execute(...)
		if ident, ok := fn.X.(*ast.Ident); ok {
			if ident.Name == "venom" {
				return true
			}
			// app.Execute(...) or any receiver
			return true
		}
		return false
	default:
		return false
	}
}

// extractFuncMeta finds a function declaration by name and extracts its metadata.
func extractFuncMeta(fset *token.FileSet, files []*ast.File, funcName string, pkgPath string) (*funcInfo, error) {
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != funcName {
				continue
			}

			info := &funcInfo{
				FuncName:    funcName,
				FullName:    pkgPath + "." + funcName,
				CommandPath: funcNameToCommandPath(funcName),
			}

			// Extract @cmd description from doc comment.
			if fn.Doc != nil {
				for _, c := range fn.Doc.List {
					annotations := parseAnnotations(c.Text)
					if desc, ok := annotations["cmd"]; ok {
						info.Description = desc
					}
				}
			}

			// Extract parameters and positional args, skipping context.Context.
			if fn.Type.Params != nil {
				extractFuncParams(fset, file, fn, info)
			}

			return info, nil
		}
	}

	return nil, fmt.Errorf("gen: function %q not found in source files", funcName)
}

// extractFuncParams extracts parameter and positional arg info from a function
// declaration, populating info.Params and info.Args. context.Context parameters
// are skipped.
func extractFuncParams(fset *token.FileSet, file *ast.File, fn *ast.FuncDecl, info *funcInfo) {
	argPos := 0

	for _, field := range fn.Type.Params.List {
		typeName := typeToString(field.Type)

		// Skip context.Context.
		if typeName == "context.Context" {
			continue
		}

		// Look for inline comment on the same line as this parameter.
		comment := findParamComment(fset, file, field)
		annotations := make(map[string]string)
		if comment != "" {
			annotations = parseAnnotations(comment)
		}

		for _, name := range field.Names {
			if _, isArg := annotations["arg"]; isArg {
				// Route to positional arg.
				a := argInfo{
					Name:     name.Name,
					Type:     typeName,
					Position: argPos,
				}
				argPos++

				// Determine cardinality.
				if strings.HasPrefix(typeName, "[]") {
					a.Cardinality = "variadic"
				} else if _, req := annotations["required"]; req {
					a.Cardinality = "required"
				} else {
					a.Cardinality = "optional"
				}

				if v, ok := annotations["default"]; ok {
					a.Default = v
				}
				if v, ok := annotations["desc"]; ok {
					a.Desc = v
				}

				info.Args = append(info.Args, a)
			} else {
				// Route to flag param (existing logic).
				p := paramInfo{
					Name:     name.Name,
					Type:     typeName,
					FlagName: paramToFlagName(name.Name),
				}

				if v, ok := annotations["short"]; ok {
					p.Short = v
				}
				if v, ok := annotations["default"]; ok {
					p.Default = v
				}
				if v, ok := annotations["desc"]; ok {
					p.Desc = v
				}
				if _, ok := annotations["required"]; ok {
					p.Required = true
				}

				info.Params = append(info.Params, p)
			}
		}
	}
}

// findParamComment returns the inline comment text for a field, if any.
func findParamComment(fset *token.FileSet, file *ast.File, field *ast.Field) string {
	// ast.Field.Comment holds the line comment (trailing comment on same line).
	if field.Comment != nil && len(field.Comment.List) > 0 {
		return field.Comment.List[0].Text
	}

	// Fallback: search file comment groups for one on the same line.
	if len(field.Names) == 0 {
		return ""
	}
	fieldLine := fset.Position(field.Pos()).Line
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if fset.Position(c.Pos()).Line == fieldLine {
				return c.Text
			}
		}
	}

	return ""
}

// typeToString renders an ast type expression as a Go type string.
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeToString(t.Elt)
		}
		return "[?]" + typeToString(t.Elt)
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// funcNameToCommandPath derives a command path from a Go function name.
// It splits on underscores into segments, then converts each segment
// from camelCase to kebab-case. A package prefix (e.g. "main.") is
// stripped first.
//
//	"serve_tls"        → ["serve", "tls"]
//	"main.serve"       → ["serve"]
//	"initProject"      → ["init-project"]
//	"init_fastProject" → ["init", "fast-project"]
func funcNameToCommandPath(name string) []string {
	// Strip package prefix if present.
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}

	if name == "" {
		return nil
	}

	parts := strings.Split(name, "_")
	// Filter out empty segments from leading/trailing/consecutive underscores,
	// and convert each segment from camelCase to kebab-case.
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, paramToFlagName(p))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// paramToFlagName converts a camelCase parameter name to kebab-case.
// This replicates the logic from the venom package.
func paramToFlagName(param string) string {
	if param == "" {
		return ""
	}

	var b strings.Builder
	runes := []rune(param)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prevLower := unicode.IsLower(runes[i-1])
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if prevLower || nextLower {
					b.WriteByte('-')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// findModulePath reads go.mod in the directory (or ancestors) and returns
// the module path and the directory containing go.mod.
func findModulePath(dir string) (string, string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", ""
	}

	d := absDir
	for {
		modFile := filepath.Join(d, "go.mod")
		data, err := os.ReadFile(modFile)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module")), d
				}
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", ""
		}
		d = parent
	}
}

// writeGenFile writes venom_gen.go to the given directory.
func writeGenFile(dir string, pkgName string, infos []*funcInfo) error {
	var b strings.Builder

	b.WriteString("// Code generated by venom; DO NOT EDIT.\n")
	fmt.Fprintf(&b, "package %s\n\n", pkgName)
	b.WriteString("import \"github.com/shakefu/venom\"\n\n")
	b.WriteString("func init() {\n")

	for _, info := range infos {
		b.WriteString("\tvenom.Register(&venom.FuncMeta{\n")
		fmt.Fprintf(&b, "\t\tFullName:    %q,\n", info.FullName)
		fmt.Fprintf(&b, "\t\tCommandPath: []string{%s},\n", formatStringSlice(info.CommandPath))
		fmt.Fprintf(&b, "\t\tDescription: %q,\n", info.Description)

		if len(info.Params) > 0 {
			b.WriteString("\t\tParams: []venom.ParamMeta{\n")
			for _, p := range info.Params {
				b.WriteString("\t\t\t{")
				fmt.Fprintf(&b, "Name: %q, Type: %q, FlagName: %q", p.Name, p.Type, p.FlagName)
				if p.Short != "" {
					fmt.Fprintf(&b, ", Short: %q", p.Short)
				}
				if p.Default != "" {
					fmt.Fprintf(&b, ", Default: %q", p.Default)
				}
				if p.Desc != "" {
					fmt.Fprintf(&b, ", Desc: %q", p.Desc)
				}
				if p.Required {
					b.WriteString(", Required: true")
				}
				b.WriteString("},\n")
			}
			b.WriteString("\t\t},\n")
		}

		if len(info.Args) > 0 {
			b.WriteString("\t\tPositionalArgs: []venom.PositionalArgMeta{\n")
			for _, a := range info.Args {
				cardConst := "venom.ArgOptional"
				switch a.Cardinality {
				case "required":
					cardConst = "venom.ArgRequired"
				case "variadic":
					cardConst = "venom.ArgVariadic"
				}
				b.WriteString("\t\t\t{")
				fmt.Fprintf(&b, "Name: %q, Type: %q, Position: %d, Cardinality: %s", a.Name, a.Type, a.Position, cardConst)
				if a.Default != "" {
					fmt.Fprintf(&b, ", Default: %q", a.Default)
				}
				if a.Desc != "" {
					fmt.Fprintf(&b, ", Desc: %q", a.Desc)
				}
				b.WriteString("},\n")
			}
			b.WriteString("\t\t},\n")
		}

		b.WriteString("\t})\n")
	}

	b.WriteString("}\n")

	// Format the generated code.
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		// If formatting fails, write the unformatted version for debugging.
		return os.WriteFile(filepath.Join(dir, "venom_gen.go"), []byte(b.String()), 0644)
	}

	return os.WriteFile(filepath.Join(dir, "venom_gen.go"), src, 0644)
}

// formatStringSlice formats a slice of strings as Go literal elements.
func formatStringSlice(ss []string) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(parts, ", ")
}
