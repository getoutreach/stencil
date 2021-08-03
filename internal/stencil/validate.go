package stencil

import (
	"regexp"
)

const ValidateNameRegexp = `^[_a-z][_a-z0-9-]*$`

///Block(global)
func ValidateName(name string) bool {
	// This is more restrictive than the actual spec.  We're artificially
	// restricting ourselves to non-Unicode names because (in practice) we
	// probably don't support international characters very well, either.
	//
	// See also:
	// 	https://golang.org/ref/spec#Identifiers
	//	https://outreach-io.atlassian.net/wiki/spaces/EN/pages/784237883/Setting+up+a+Go+repo
	acceptableName := regexp.MustCompile(ValidateNameRegexp)
	return acceptableName.MatchString(name)
}

///EndBlock(global)
