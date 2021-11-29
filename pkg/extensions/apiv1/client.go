package apiv1

import (
	"os"

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
func NewExtensionImplementation(impl Implementation) error {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: false,
	})

	plugin.Serve(&plugin.ServeConfig{
		Logger:          logger,
		HandshakeConfig: NewHandshake(),
		Plugins: map[string]plugin.Plugin{
			Name: &ExtensionPlugin{Impl: impl},
		},
	})

	return nil
}
