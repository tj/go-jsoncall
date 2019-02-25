package jsoncall

import (
	"context"
	"reflect"
)

// errorInterface is the error interface.
var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

// contextInterface is the context interface.
var contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()

// typeName returns the JSON name of the corresponding Go type.
func typeName(t reflect.Type) string {
	switch unrollPointer(t).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array of " + typeName(t.Elem()) + "s"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Map:
		return "object"
	case reflect.Struct:
		return "object"
	default:
		return "unknown"
	}
}

// unrollPointer unrolls and pointers of a type.
func unrollPointer(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// hasContext returns true if the function type has a context argument at the given index.
func hasContext(t reflect.Type, i int) bool {
	if t.NumIn() < i+1 {
		return false
	}

	return isContext(t.In(i))
}

// isContext returns true if the given type implements context.Context.
func isContext(t reflect.Type) bool {
	return t.Kind() == reflect.Interface && t.Implements(contextInterface)
}

// isError returns true if the given type implements error.
func isError(t reflect.Type) bool {
	return t.Kind() == reflect.Interface && t.Implements(errorInterface)
}
