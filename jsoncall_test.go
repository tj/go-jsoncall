package jsoncall_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/tj/assert"
	jsoncall "github.com/tj/go-jsoncall"
)

func abs(v float64) float64 {
	return math.Abs(v)
}

func add(a, b int) int {
	return a + b
}

func sum(nums ...int) (sum int) {
	for _, n := range nums {
		sum += n
	}
	return
}

func avg(nums ...int) int {
	return sum(nums...) / len(nums)
}

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func addUser(u User) error {
	return nil
}

func addUserPointer(u *User) error {
	return nil
}

func addUsers(u []User) error {
	return nil
}

func addUserContext(ctx context.Context, u User) error {
	return nil
}

func addPet(name string) error {
	return errors.New("error adding pet")
}

type mathService struct{}

func (m *mathService) Sum(ctx context.Context, nums []int) int {
	return sum(nums...)
}

// Test normalization of arguments.
func TestNormalize(t *testing.T) {
	assert.Equal(t, `[]`, jsoncall.Normalize(``))
	assert.Equal(t, `[]`, jsoncall.Normalize(`[]`))
	assert.Equal(t, `[5]`, jsoncall.Normalize(`5`))
	assert.Equal(t, `[5]`, jsoncall.Normalize(`  5`))
	assert.Equal(t, `["Hello"]`, jsoncall.Normalize(`"Hello"`))
	assert.Equal(t, `["Hello"]`, jsoncall.Normalize(`  "Hello"  `))
	assert.Equal(t, `[{ "name": "Tobi" }]`, jsoncall.Normalize(`{ "name": "Tobi" }`))
	assert.Equal(t, `[1, 2, 3]`, jsoncall.Normalize(`[1, 2, 3]`))
}

// Test arguments from a function signature.
func TestArgumentsOfFunc(t *testing.T) {
	t.Run("should support no results", func(t *testing.T) {
		noop := func() {}
		vals, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(noop), `[]`)
		assert.NoError(t, err)
		assert.Len(t, vals, 0)
	})

	t.Run("should error when the input is invalid json", func(t *testing.T) {
		_, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(add), `[5, hey]`)
		assert.EqualError(t, err, `Invalid JSON`)
	})

	t.Run("should error when too few arguments are passed", func(t *testing.T) {
		_, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(add), `[1]`)
		assert.EqualError(t, err, `Too few arguments passed`)
	})

	t.Run("should error when too many arguments are passed", func(t *testing.T) {
		_, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(add), `[1, 2, 3]`)
		assert.EqualError(t, err, `Too many arguments passed`)
	})

	t.Run("should error when arguments are incorrect types", func(t *testing.T) {
		_, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(add), `[1, "5"]`)
		assert.EqualError(t, err, `Incorrect type string, expected number`)

		_, err = jsoncall.ArgumentsOfFunc(reflect.TypeOf(addUser), `["hello"]`)
		assert.EqualError(t, err, `Incorrect type string, expected object`)
	})

	t.Run("should support primitives", func(t *testing.T) {
		vals, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(add), `[1, 5]`)
		assert.NoError(t, err)
		assert.Len(t, vals, 2)
		assert.Equal(t, 1, vals[0].Interface().(int))
		assert.Equal(t, 5, vals[1].Interface().(int))
	})

	t.Run("should support structs", func(t *testing.T) {
		vals, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(addUser), `[{ "name": "Tobi" }]`)
		assert.NoError(t, err)
		assert.Len(t, vals, 1)
		assert.Equal(t, "Tobi", vals[0].Interface().(User).Name)
	})

	t.Run("should support pointers to structs", func(t *testing.T) {
		vals, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(addUserPointer), `[{ "name": "Tobi" }]`)
		assert.NoError(t, err)
		assert.Len(t, vals, 1)
		assert.Equal(t, "Tobi", vals[0].Interface().(*User).Name)
	})

	t.Run("should support null for pointers to structs", func(t *testing.T) {
		vals, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(addUserPointer), `[null]`)
		assert.NoError(t, err)
		assert.Len(t, vals, 1)
		assert.Empty(t, vals[0].Interface())
	})

	t.Run("should support context as the first argument", func(t *testing.T) {
		vals, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(addUserContext), `[{ "name": "Tobi" }]`)
		assert.NoError(t, err)
		assert.Len(t, vals, 2)
		assert.Implements(t, (*context.Context)(nil), vals[0].Interface(), "should have a context")
		assert.Equal(t, "Tobi", vals[1].Interface().(User).Name)
	})

	t.Run("should support custom contexts via WithContextFunc", func(t *testing.T) {
		var called bool
		_, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(addUserContext), `[{ "name": "Tobi" }]`, jsoncall.WithContextFunc(func() context.Context {
			called = true
			return context.TODO()
		}))
		assert.NoError(t, err)
		assert.True(t, called, "should call the function")
	})

	t.Run("should support slices of structs", func(t *testing.T) {
		vals, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(addUsers), `[[{ "name": "Tobi" }, { "name": "Loki" }]]`)
		assert.NoError(t, err)
		assert.Len(t, vals, 1)
		assert.Equal(t, "Tobi", vals[0].Interface().([]User)[0].Name)
	})

	t.Run("should error on variadic functions", func(t *testing.T) {
		// TODO: support variadic functions
		_, err := jsoncall.ArgumentsOfFunc(reflect.TypeOf(sum), `[1, 2, 3, 4]`)
		assert.EqualError(t, err, `Variadic functions are not yet supported`)
	})
}

// Test arguments from a method signature.
func TestArgumentsOfMethod(t *testing.T) {
	s := &mathService{}

	m, ok := reflect.TypeOf(s).MethodByName("Sum")
	assert.True(t, ok)

	vals, err := jsoncall.ArgumentsOfMethod(m, `[[1,2,3,4]]`)
	assert.NoError(t, err)
	assert.Implements(t, (*context.Context)(nil), vals[0].Interface(), "should have a context")
	assert.Equal(t, []int{1, 2, 3, 4}, vals[1].Interface())
}

// Test calling of functions.
func TestCallFunc(t *testing.T) {
	t.Run("should support returning a value", func(t *testing.T) {
		add := func(a, b int) int { return a + b }
		v, err := jsoncall.CallFunc(add, `[1,2]`)
		assert.NoError(t, err)
		assert.Len(t, v, 1)
		assert.Equal(t, 3, v[0].Interface())
	})

	t.Run("should support returning errors", func(t *testing.T) {
		add := func(a, b int) error { return errors.New("boom") }
		_, err := jsoncall.CallFunc(add, `[1,2]`)
		assert.EqualError(t, err, `boom`)
	})

	t.Run("should support returning values and errors", func(t *testing.T) {
		add := func(a, b int) (int, error) { return a + b, nil }
		v, err := jsoncall.CallFunc(add, `[1,2]`)
		assert.NoError(t, err)
		assert.Equal(t, 3, v[0].Interface())
	})

	t.Run("should support returning multiple values", func(t *testing.T) {
		minmax := func(a, b int) (min, max int) { return a, b }
		v, err := jsoncall.CallFunc(minmax, `[1,2]`)
		assert.NoError(t, err)
		assert.Equal(t, 1, v[0].Interface())
		assert.Equal(t, 2, v[1].Interface())
	})

	t.Run("should support returning multiple values and errors", func(t *testing.T) {
		minmax := func(a, b int) (min, max int, err error) { return a, b, nil }
		v, err := jsoncall.CallFunc(minmax, `[1,2]`)
		assert.NoError(t, err)
		assert.Equal(t, 1, v[0].Interface())
		assert.Equal(t, 2, v[1].Interface())
	})

	t.Run("should support returning errors", func(t *testing.T) {
		_, err := jsoncall.CallFunc(addPet, `["Tobi"]`)
		assert.EqualError(t, err, `error adding pet`)
	})
}

// Test calling of methods.
func TestCallMethod(t *testing.T) {
	t.Run("should support returning a value", func(t *testing.T) {
		s := &mathService{}
		m, _ := reflect.TypeOf(s).MethodByName("Sum")
		v, err := jsoncall.CallMethod(s, m, `[[1,2]]`)
		assert.NoError(t, err)
		assert.Len(t, v, 1)
		assert.Equal(t, 3, v[0].Interface())
	})
}

// Benchmark argument reflection.
func BenchmarkArguments(b *testing.B) {
	b.SetBytes(1)
	t := reflect.TypeOf(addUsers)
	for i := 0; i < b.N; i++ {
		jsoncall.ArgumentsOfFunc(t, `[[{ "name": "Tobi" }, { "name": "Loki" }]]`)
	}
}

// Benchmark function calling.
func BenchmarkCallFunc(b *testing.B) {
	b.SetBytes(1)
	for i := 0; i < b.N; i++ {
		jsoncall.CallFunc(addUser, `[[{ "name": "Tobi" }, { "name": "Loki" }]]`)
	}
}
