// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file resolves the version of a git repository, with retries.

package git

import (
	"context"

	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	"github.com/getoutreach/stencil/internal/retry"
	"github.com/sirupsen/logrus"
)

// ResolveVersion is resolver.Resolve, retried (3 attempts by default) when the
// remote fails in a way that could go away, see IsTransientError. A version that
// does not exist or a constraint that nothing satisfies is not retried. opts are
// applied after the defaults.
func ResolveVersion(ctx context.Context, log logrus.FieldLogger, token cfg.SecretData, c resolver.Criteria,
	opts ...retry.Option) (*resolver.Version, error) {
	opts = append([]retry.Option{
		retry.WithRetryable(IsTransientError),
		retry.WithLogging(log, "resolve "+RedactURL(c.URL)),
	}, opts...)

	return retry.DoValue(ctx, func(ctx context.Context) (*resolver.Version, error) {
		// Resolve sets the channel and the constraints of the criteria that it is
		// given, so each attempt gets a copy.
		attempt := c
		v, err := resolver.Resolve(ctx, token, &attempt)
		return v, RedactError(err)
	}, opts...)
}
