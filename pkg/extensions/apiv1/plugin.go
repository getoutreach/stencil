// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the plugin RPC logic for the extension host

package apiv1

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// _ is a compile time assertion we implement the interface
var _ Implementation = &ExtensionPluginClient{}

// ExtensionPluginClient implements a client that communicates over RPC
type ExtensionPluginClient struct{ client *rpc.Client }

// GetConfig returns the config for the extension
func (g *ExtensionPluginClient) GetConfig() (*Config, error) {
	var resp *Config
	err := g.client.Call("Plugin.GetConfig", new(interface{}), &resp)
	return resp, err
}

// GetTemplateFunctions returns the template functions for this extension
func (g *ExtensionPluginClient) GetTemplateFunctions() ([]*TemplateFunction, error) {
	var resp []*TemplateFunction
	err := g.client.Call("Plugin.GetTemplateFunctions", new(interface{}), &resp)
	return resp, err
}

// ExecuteTemplateFunction exectues a template function for this extension
func (g *ExtensionPluginClient) ExecuteTemplateFunction(t *TemplateFunctionExec) (interface{}, error) {
	var resp interface{}
	err := g.client.Call("Plugin.ExecuteTemplateFunction", t, &resp)
	return resp, err
}

// ExtensionPluginServer implements a rpc compliant server
type ExtensionPluginServer struct {
	Impl Implementation
}

// GetConfig returns the config for this extension
func (s *ExtensionPluginServer) GetConfig(args interface{}, resp **Config) error {
	v, err := s.Impl.GetConfig()
	*resp = v
	return err
}

// GetTemplateFunctions returns the template functions for this extension
func (s *ExtensionPluginServer) GetTemplateFunctions(args interface{}, resp *[]*TemplateFunction) error {
	v, err := s.Impl.GetTemplateFunctions()
	*resp = v
	return err
}

// ExecuteTemplateFunction executes a template function for this extension
//nolint:gocritic // Why: go-plugin wants this
func (s *ExtensionPluginServer) ExecuteTemplateFunction(t *TemplateFunctionExec, resp *interface{}) error {
	v, err := s.Impl.ExecuteTemplateFunction(t)
	*resp = v
	return err
}

// ExtensionPlugin is the high level plugin used by go-plugin
// it stores both the server and client implementation
type ExtensionPlugin struct {
	// Impl is the real implementation for this extension
	Impl Implementation
}

// Server is a extension server
func (p *ExtensionPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ExtensionPluginServer{Impl: p.Impl}, nil
}

// Client is a extension client
func (*ExtensionPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ExtensionPluginClient{client: c}, nil
}
