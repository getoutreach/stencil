package apiv1

import (
	"net/rpc"
)

// _ is a compile time assertion we implement the interface
var _ implementationTransport = &underlyingPluginClient{}

// underlyingPluginClient implements the plugin client over
// rpc. This is a low level interface responsible for transmitting
// the implementationTransport over the wire.
type underlyingPluginClient struct{ client *rpc.Client }

// GetConfig returns the config for the extension
func (g *underlyingPluginClient) GetConfig() (*Config, error) {
	var resp *Config
	err := g.client.Call("Plugin.GetConfig", new(interface{}), &resp)
	return resp, err
}

// GetTemplateFunctions returns the template functions for this extension
func (g *underlyingPluginClient) GetTemplateFunctions() ([]*TemplateFunction, error) {
	var resp []*TemplateFunction
	err := g.client.Call("Plugin.GetTemplateFunctions", new(interface{}), &resp)
	return resp, err
}

// ExecuteTemplateFunction exectues a template function for this extension
func (g *underlyingPluginClient) ExecuteTemplateFunction(t *TemplateFunctionExec) ([]byte, error) {
	// IDEA(jaredallard): Actually stream this data in the future
	var resp []byte
	err := g.client.Call("Plugin.ExecuteTemplateFunction", t, &resp)
	return resp, err
}
