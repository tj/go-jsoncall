// Package jsoncall provides utilities for invoking Go functions from JSON.
package jsoncall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// config settings.
type config struct {
	contextFunc  ContextFunc
	arity        int
	offset       int
	contextIndex int
}

// defaultContextFunc is the default context function.
func defaultContextFunc() context.Context {
	return context.Background()
}

// ErrNotFunction is returned when a non-function value is passed.
var ErrNotFunction = errors.New("Must pass a function")

// ErrTooManyArguments is returned when too many arguments are passed.
var ErrTooManyArguments = errors.New("Too many arguments passed")

// ErrTooFewArguments is returned when too few arguments are passed.
var ErrTooFewArguments = errors.New("Too few arguments passed")

// ErrInvalidJSON is returned when the input is malformed.
var ErrInvalidJSON = errors.New("Invalid JSON")

// errVariadic is returned when a variadic function is used.
var errVariadic = errors.New("Variadic functions are not yet supported")

// UnmarshalError is an unmarshal error.
type UnmarshalError json.UnmarshalTypeError

// Error implementation.
func (e UnmarshalError) Error() string {
	return fmt.Sprintf("Incorrect type %s, expected %s", e.Value, typeName(e.Type))
}

// ContextFunc is used to create a new context.
type ContextFunc func() context.Context

// Option function.
type Option func(*config)

// WithContextFunc sets the context function, used to create a new context
// when the function being called expects one.
func WithContextFunc(fn ContextFunc) Option {
	return func(v *config) {
		v.contextFunc = fn
	}
}

// newConfig returns a new config with options applied.
func newConfig(options []Option) *config {
	var c config
	c.contextFunc = defaultContextFunc
	for _, o := range options {
		o(&c)
	}
	return &c
}

// Normalize returns a normalized json array string, to be used as parameters.
func Normalize(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 0 && s[0] == '[' {
		return s
	}
	return "[" + s + "]"
}

// CallFunc invokes a function with arguments derived from a json string.
func CallFunc(fn interface{}, args string, options ...Option) ([]reflect.Value, error) {
	t := reflect.TypeOf(fn)

	arguments, err := ArgumentsOfFunc(t, args, options...)
	if err != nil {
		return nil, err
	}

	return CallFuncArgs(fn, arguments, options...)
}

// CallMethod invokes a method on a struct with arguments derived from a json string.
func CallMethod(receiver interface{}, m reflect.Method, args string, options ...Option) ([]reflect.Value, error) {
	arguments, err := ArgumentsOfMethod(m, args, options...)
	if err != nil {
		return nil, err
	}

	return CallMethodArgs(receiver, m, arguments, options...)
}

// CallFuncArgs invokes a function with arguments derived from a json string.
func CallFuncArgs(fn interface{}, args []reflect.Value, options ...Option) (values []reflect.Value, err error) {
	// invoke
	res := reflect.ValueOf(fn).Call(args)

	// results
	for _, v := range res {
		if isError(v.Type()) && v.IsValid() && !v.IsNil() {
			return nil, v.Interface().(error)
		}
		values = append(values, v)
	}

	return
}

// CallMethodArgs invokes a method on a struct with arguments derived from a json string.
func CallMethodArgs(receiver interface{}, m reflect.Method, args []reflect.Value, options ...Option) (values []reflect.Value, err error) {
	// receiver
	r := reflect.ValueOf(receiver)
	args = append([]reflect.Value{r}, args...)

	// invoke
	res := m.Func.Call(args)

	// results
	for _, v := range res {
		if isError(v.Type()) && v.IsValid() && !v.IsNil() {
			return nil, v.Interface().(error)
		}
		values = append(values, v)
	}

	return
}

// ArgumentsOfMethod returns arguments for the given method, derived from a json string.
func ArgumentsOfMethod(m reflect.Method, args string, options ...Option) ([]reflect.Value, error) {
	c := newConfig(options)
	c.arity = m.Type.NumIn() - 1
	c.offset = 1
	c.contextIndex = 1
	return arguments(m.Type, args, c)
}

// ArgumentsOfFunc returns arguments for the given function, derived from a json string.
func ArgumentsOfFunc(t reflect.Type, args string, options ...Option) ([]reflect.Value, error) {
	if t.Kind() != reflect.Func {
		return nil, ErrNotFunction
	}
	c := newConfig(options)
	c.arity = t.NumIn()
	return arguments(t, args, c)
}

// arguments implementation.
func arguments(t reflect.Type, s string, c *config) ([]reflect.Value, error) {
	var args []reflect.Value

	// ensure it's not variadic
	if t.IsVariadic() {
		return nil, errVariadic
	}

	// inject context
	if hasContext(t, c.contextIndex) {
		args = append(args, reflect.ValueOf(c.contextFunc()))
		c.offset++
		c.arity--
	}

	// parse params
	var params []json.RawMessage

	err := json.Unmarshal([]byte(s), &params)

	if _, ok := err.(*json.SyntaxError); ok {
		return nil, ErrInvalidJSON
	}

	if err != nil {
		return nil, err
	}

	// too few
	if len(params) < c.arity {
		return nil, ErrTooFewArguments
	}

	// too many
	if len(params) > c.arity {
		return nil, ErrTooManyArguments
	}

	// process the arguments
	for i := 0; i < c.arity; i++ {
		kind := t.In(c.offset + i)
		arg := reflect.New(kind)
		value := arg.Interface()

		err := json.Unmarshal(params[i], value)

		if e, ok := err.(*json.UnmarshalTypeError); ok {
			return nil, UnmarshalError(*e)
		}

		if err != nil {
			return nil, err
		}

		args = append(args, arg.Elem())
	}

	return args, nil
}
