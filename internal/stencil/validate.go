package stencil

import (
	"regexp"
)

// ValidateNameRegexp is the regex used to validate the service's name
const ValidateNameRegexp = `^[_a-z][_a-z0-9-]*$`

// ValidateName ensures that the name of a service in the manifest
// fits the criteria we require.
func ValidateName(name string) bool {
	// This is more restrictive than the actual spec.  We're artificially
	// restricting ourselves to non-Unicode names because (in practice) we
	// probably don't support international characters very well, either.
	//
	// See also:
	// 	https://golang.org/ref/spec#Identifiers
	acceptableName := regexp.MustCompile(ValidateNameRegexp)
	return acceptableName.MatchString(name)
}
