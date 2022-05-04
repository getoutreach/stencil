// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements the plugin client (stencil -> plugin)

package apiv1

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// IDEA(jaredallard): Cleanup this to return a Implementation backed by a transport as well.

// NewExtensionClient creates a new Implementation from a plugin
func NewExtensionClient(ctx context.Context, extPath string, log logrus.FieldLogger) (Implementation, error) {
	// create a connection to the extension
	client := plugin.NewClient(&plugin.ClientConfig{
		Logger: hclog.New(&hclog.LoggerOptions{
			Level:       hclog.Trace,
			Output:      &logger{fn: log.Debug},
			DisableTime: true,
		}),
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  Version,
			MagicCookieKey:   CookieKey,
			MagicCookieValue: CookieValue,
		},
		Plugins: map[string]plugin.Plugin{
			Name: &ExtensionPlugin{log, nil},
		},
		Cmd: exec.CommandContext(ctx, extPath),
	})

	rpcClient, err := client.Client()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create connection to extension")
	}

	raw, err := rpcClient.Dispense(Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup extension connection over extension")
	}

	ext, ok := raw.(implementationTransport)
	if !ok {
		return nil, fmt.Errorf("failed to create apiv1.Implementation from type %s", reflect.TypeOf(raw).String())
	}

	return newImplementationTransportToImplementation(ext), nil
}
