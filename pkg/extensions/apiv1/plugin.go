// Copyright 2021 Outreach Corporation. All Rights Reserved.

// Description: Implements the plugin RPC logic for the extension host

package apiv1

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// compile time assertion we implement the interface
var _ Implementation = &ExtensionPluginClient{}

// ExtensionPluginClient implements a client that communicates over RPC
type ExtensionPluginClient struct{ client *rpc.Client }

func (g *ExtensionPluginClient) GetConfig() (*Config, error) {
	var resp *Config
	err := g.client.Call("Plugin.GetConfig", new(interface{}), &resp)
	return resp, err
}

func (g *ExtensionPluginClient) GetTemplateFunctions() ([]*TemplateFunction, error) {
	var resp []*TemplateFunction
	err := g.client.Call("Plugin.GetTemplateFunctions", new(interface{}), &resp)
	return resp, err
}

func (g *ExtensionPluginClient) ExecuteTemplateFunction(t *TemplateFunctionExec) (interface{}, error) {
	var resp interface{}
	err := g.client.Call("Plugin.ExecuteTemplateFunction", t, &resp)
	return resp, err
}

// ExtensionPluginServer implements a rpc compliant server
type ExtensionPluginServer struct {
	Impl Implementation
}

func (s *ExtensionPluginServer) GetConfig(args interface{}, resp **Config) error {
	v, err := s.Impl.GetConfig()
	*resp = v
	return err
}

func (s *ExtensionPluginServer) GetTemplateFunctions(args interface{}, resp *[]*TemplateFunction) error {
	v, err := s.Impl.GetTemplateFunctions()
	*resp = v
	return err
}

//nolint:gocritic // Why: This is how go-plugin does it. :shrug:
func (s *ExtensionPluginServer) ExecuteTemplateFunction(t *TemplateFunctionExec, resp *interface{}) error {
	v, err := s.Impl.ExecuteTemplateFunction(t)
	*resp = v
	return err
}

// ExtensionPlugin is the high level plugin used by go-plugin
// it stores both the server and client implementation
type ExtensionPlugin struct {
	Impl Implementation
}

func (p *ExtensionPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ExtensionPluginServer{Impl: p.Impl}, nil
}

func (*ExtensionPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ExtensionPluginClient{client: c}, nil
}
