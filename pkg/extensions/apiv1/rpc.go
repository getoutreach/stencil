// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the plugin RPC logic for the extension host

package apiv1

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"
)

// ExtensionPlugin is the high level plugin used by go-plugin
// it stores both the server and client implementation
type ExtensionPlugin struct {
	log  logrus.FieldLogger
	impl implementationTransport
}

// Server serves a implementationTransport over net/rpc
func (p *ExtensionPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &rpcTransportServer{p.log, p.impl}, nil
}

// Client serves a Implementation over net/rpc
func (p *ExtensionPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &rpcTransportClient{p.log, c}, nil
}
