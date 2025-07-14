// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the functions file

package codegen

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

func Example_toYaml() {
	example := map[string]interface{}{
		"a": "b",
		"c": "d",
	}
	fmt.Println(toYAML(example))

	// Output:
	// a: b
	// c: d <nil>
}

func Example_fromYaml() {
	example := `
a: b
c: d
`
	fmt.Println(fromYAML(example))

	// Output:
	// map[a:b c:d] <nil>
}

func Example_toJson() {
	example := map[string]interface{}{
		"a": "b",
		"c": "d",
	}
	fmt.Println(toJSON(example))

	// Output:
	// {"a":"b","c":"d"} <nil>
}

func Example_fromJson() {
	example := `
{
	"a": "b",
	"c": "d"
}
`
	fmt.Println(fromJSON(example))

	// Output:
	// map[a:b c:d] <nil>
}

func Example_toTOML() {
	example := map[string]any{
		"a": "b",
		"c": "d",
	}
	fmt.Println(toTOML(example))

	// Output:
	// a = "b"
	// c = "d" <nil>
}

func Example_fromTOML() {
	example := `a = "b"
c = "d"
`
	fmt.Println(fromTOML(example))
	// Output:
	// map[a:b c:d] <nil>
}
