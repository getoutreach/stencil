package apiv1

// underlyingPluginServer implements a rpc backed implementatio
// of implementationTransport.
type underlyingPluginServer struct {
	impl implementationTransport
}

// GetConfig returns the config for this extension
func (s *underlyingPluginServer) GetConfig(args interface{}, resp **Config) error {
	v, err := s.impl.GetConfig()
	*resp = v
	return err
}

// GetTemplateFunctions returns the template functions for this extension
func (s *underlyingPluginServer) GetTemplateFunctions(args interface{}, resp *[]*TemplateFunction) error {
	v, err := s.impl.GetTemplateFunctions()
	*resp = v
	return err
}

// ExecuteTemplateFunction executes a template function for this extension
//nolint:gocritic // Why: go-plugin wants this
func (s *underlyingPluginServer) ExecuteTemplateFunction(t *TemplateFunctionExec, resp *[]byte) error {
	v, err := s.impl.ExecuteTemplateFunction(t)
	*resp = v
	return err
}
