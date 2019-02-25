package jsoncall

import (
	"reflect"
	"testing"

	"github.com/tj/assert"
)

// Test type name conversion.
func TestTypeName(t *testing.T) {
	cases := []struct {
		input  interface{}
		output string
	}{
		{1, "number"},
		{1.5, "number"},
		{"hello", "string"},
		{true, "boolean"},
		{struct{}{}, "object"},
		{map[string]string{}, "object"},
		{[]string{}, "array of strings"},
		{[]bool{}, "array of booleans"},
		{[]int{}, "array of numbers"},
	}

	for _, c := range cases {
		t.Run(c.output, func(t *testing.T) {
			v := typeName(reflect.TypeOf(c.input))
			assert.Equal(t, c.output, v)
		})
	}
}
