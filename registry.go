package venom

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
)

var (
	registry = map[string]*FuncMeta{}
	mu       sync.Mutex
)

// Register stores function metadata in the global registry, keyed by FullName.
// Called by generated init() code.
func Register(meta *FuncMeta) {
	mu.Lock()
	defer mu.Unlock()
	registry[meta.FullName] = meta
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
