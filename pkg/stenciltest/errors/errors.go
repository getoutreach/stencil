// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains errors returned by stenciltest

// Package errors contains errors returned by stenciltest.
package errors

import (
	"fmt"

	pkgerrors "github.com/pkg/errors"
)

// Wrap wraps an error with a stenciltest prefix
func Wrap(err error, msg string) error {
	return pkgerrors.Wrap(err, WrapString(msg))
}

// WrapString wraps an error with a stenciltest prefix
func WrapString(msg string) string {
	return pkgerrors.Wrap(fmt.Errorf("%s", msg), "stenciltest").Error()
}
