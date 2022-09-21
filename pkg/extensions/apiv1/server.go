// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements a plugin Implementation
// for the extensions host.

package apiv1

import (
	"github.com/getoutreach/stencil/internal/log"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// NewHandshake returns a plugin.HandshakeConfig for
// this extension api version.
func NewHandshake() plugin.HandshakeConfig {
	return plugin.HandshakeConfig{
		ProtocolVersion:  Version,
		MagicCookieKey:   CookieKey,
		MagicCookieValue: CookieValue,
	}
}

// NewExtensionImplementation implements a new extension
// and starts serving it.
func NewExtensionImplementation(impl Implementation, log log.Logger) error {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:       hclog.Trace,
		Output:      &logger{fn: log.Debug},
		DisableTime: true,
	})

	plugin.Serve(&plugin.ServeConfig{
		Logger:          logger,
		HandshakeConfig: NewHandshake(),
		Plugins: map[string]plugin.Plugin{
			Name: &ExtensionPlugin{log, newImplementationToImplementationTransport(impl)},
		},
	})

	return nil
}
