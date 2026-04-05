package venom

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"sync"
)

var (
	registry = map[string]*FuncMeta{}
	mu       sync.Mutex
)

// Register stores function metadata in the global registry, keyed by FullName.
// Called by generated init() code.
func Register(meta *FuncMeta) {
	validatePositionalArgs(meta)
	mu.Lock()
	defer mu.Unlock()
	registry[meta.FullName] = meta
}

// validatePositionalArgs enforces the five positional-argument invariants
// defined in the Allium spec. Panics if any invariant is violated.
func validatePositionalArgs(meta *FuncMeta) {
	args := meta.PositionalArgs
	if len(args) == 0 {
		return
	}

	// Sort a copy by position for ordered checks.
	sorted := make([]PositionalArgMeta, len(args))
	copy(sorted, args)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	// UniquePositionalPositions — no two positional args share the same position.
	seen := make(map[int]string, len(sorted))
	for _, a := range sorted {
		if prev, dup := seen[a.Position]; dup {
			panic(fmt.Sprintf("venom: %s: positional args %q and %q share position %d",
				meta.FullName, prev, a.Name, a.Position))
		}
		seen[a.Position] = a.Name
	}

	// AtMostOneVariadic — at most one variadic arg per command.
	var variadics []string
	for _, a := range sorted {
		if a.Cardinality == ArgVariadic {
			variadics = append(variadics, a.Name)
		}
	}
	if len(variadics) > 1 {
		panic(fmt.Sprintf("venom: %s: multiple variadic positional args: %v",
			meta.FullName, variadics))
	}

	// RequiredBeforeOptional — all required args must precede any optional ones.
	seenOptional := false
	for _, a := range sorted {
		if a.Cardinality == ArgOptional {
			seenOptional = true
		}
		if a.Cardinality == ArgRequired && seenOptional {
			panic(fmt.Sprintf("venom: %s: required positional arg %q (position %d) follows an optional arg",
				meta.FullName, a.Name, a.Position))
		}
	}

	// VariadicIsLast — a variadic arg must have the highest position.
	if len(variadics) == 1 {
		var variadicPos int
		var maxNonVariadicPos int
		var maxNonVariadicName string
		for _, a := range sorted {
			if a.Cardinality == ArgVariadic {
				variadicPos = a.Position
			} else if a.Position > maxNonVariadicPos {
				maxNonVariadicPos = a.Position
				maxNonVariadicName = a.Name
			}
		}
		if maxNonVariadicName != "" && variadicPos <= maxNonVariadicPos {
			panic(fmt.Sprintf("venom: %s: variadic arg %q (position %d) must be after all other positional args (e.g. %q at position %d)",
				meta.FullName, variadics[0], variadicPos, maxNonVariadicName, maxNonVariadicPos))
		}
	}

	// VariadicIsSlice — variadic args must have type "[]string".
	for _, a := range sorted {
		if a.Cardinality == ArgVariadic && a.Type != "[]string" {
			panic(fmt.Sprintf("venom: %s: variadic positional arg %q must have type []string, got %q",
				meta.FullName, a.Name, a.Type))
		}
	}
}

// funcName extracts the runtime-qualified name from a function value using
// reflection. Panics if fn is not a function or the name cannot be resolved.
func funcName(fn interface{}) string {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		panic(fmt.Sprintf("venom: expected a function, got %T", fn))
	}
	pc := v.Pointer()
	f := runtime.FuncForPC(pc)
	if f == nil {
		panic(fmt.Sprintf("venom: cannot resolve function name for %T", fn))
	}
	return f.Name()
}

// lookupMeta finds registered metadata for a function value. Returns an error
// if the function has not been registered (typically meaning codegen has not
// been run).
func lookupMeta(fn interface{}) (*FuncMeta, error) {
	name := funcName(fn) // panics on non-function
	mu.Lock()
	defer mu.Unlock()
	meta, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("venom: no metadata registered for %s; did you run go generate?", name)
	}
	meta.Func = fn
	return meta, nil
}
