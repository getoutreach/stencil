package functions

import (
	"fmt"
	"reflect"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDereference(t *testing.T) {
	str := "abcdef"

	resp := dereference(str)
	assert.Equal(t, reflect.TypeOf(resp).Kind(), reflect.String, "expected dereference() to return non-ptr type")
	assert.Equal(t, resp.(string), str, "expected dereference() to not corrupt value")

	resp = dereference(&str)
	assert.Equal(t, reflect.TypeOf(resp).Kind(), reflect.String, "expected dereference() to return ptr type as non-pointer")
	assert.Equal(t, resp.(string), str, "expected dereference() to not corrupt value")
}

func Example_quotejoinstrings() {
	example := []string{"a", "b", "c"}
	fmt.Println(quotejoinstrings(example, " "))

	// Output:
	// "a" "b" "c"
}
