// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the token that resolves modules.

package modules

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/getoutreach/gobox/pkg/cfg"
	"gotest.tools/v3/assert"
)

var errNoToken = errors.New("no token")

func TestResolveToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts *ModuleResolveOptions
		want cfg.SecretData
	}{
		{name: "Token is used without TokenFunc", opts: &ModuleResolveOptions{Token: "plain"}, want: "plain"},
		{
			name: "TokenFunc takes the place of Token",
			opts: &ModuleResolveOptions{
				Token:     "plain",
				TokenFunc: func() (cfg.SecretData, error) { return "from func", nil },
			},
			want: "from func",
		},
		{
			name: "an error from TokenFunc leaves the token empty",
			opts: &ModuleResolveOptions{
				Token:     "plain",
				TokenFunc: func() (cfg.SecretData, error) { return "", errNoToken },
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.opts.resolveToken(), tt.want)
		})
	}
}

func TestResolveTokenCallsTokenFuncOnce(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	opts := &ModuleResolveOptions{TokenFunc: func() (cfg.SecretData, error) {
		calls.Add(1)
		return "token", nil
	}}

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { assert.Equal(t, opts.resolveToken(), cfg.SecretData("token")) })
	}
	wg.Wait()

	assert.Equal(t, calls.Load(), int32(1))
}
