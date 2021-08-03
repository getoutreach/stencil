package functions

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

// Default are stock template functions that don't impact
// the generation of a file. Anything that does that should be located
// in the scope of the file renderer function instead
var Default = template.FuncMap{
	"Dereference": func(i interface{}) interface{} {
		infType := reflect.TypeOf(i)

		// If not a pointer, noop
		if infType.Kind() != reflect.Ptr {
			return i
		}

		return reflect.ValueOf(i).Elem().Interface()
	},
	"QuoteJoinStrings": func(elems []string, sep string) string {
		for i := range elems {
			elems[i] = fmt.Sprintf("\"%s\"", elems[i])
		}
		return strings.Join(elems, sep)
	},
}
