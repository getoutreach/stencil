// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file removes credentials from URLs that are logged.

package git

import "regexp"

var userinfoPattern = regexp.MustCompile(`://[^/?#]*@`)

// RedactURL returns rawURL without its user information, so that it can be logged.
// A token is often written there, as in https://TOKEN@github.com/org/repo. It does
// not cover a token in the query of a URL, or in an scp-style address.
func RedactURL(rawURL string) string {
	return userinfoPattern.ReplaceAllString(rawURL, "://")
}

type redactedError struct{ err error }

func (e redactedError) Error() string { return RedactURL(e.err.Error()) }
func (e redactedError) Unwrap() error { return e.err }

// RedactError returns err with RedactURL applied to its message, so that it can be
// logged or shown. The errors of requests to a remote often contain the URL.
// errors.Is and errors.As still see the original error, which keeps its URL.
func RedactError(err error) error {
	if err == nil {
		return nil
	}
	return redactedError{err}
}
