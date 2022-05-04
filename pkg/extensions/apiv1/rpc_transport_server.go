// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements the rpc server transport for go-plugin

package apiv1

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
)

// rpcTransportServer implements a rpc backed implementation
// of implementationTransport.
type rpcTransportServer struct {
	log  logrus.FieldLogger
	impl implementationTransport
}

// GetConfig returns the config for this extension
func (s *rpcTransportServer) GetConfig(args interface{}, resp **Config) error {
	v, err := s.impl.GetConfig()
	*resp = v
	return err
}

// GetTemplateFunctions returns the template functions for this extension
func (s *rpcTransportServer) GetTemplateFunctions(args interface{}, resp *[]*TemplateFunction) error {
	v, err := s.impl.GetTemplateFunctions()
	*resp = v
	return err
}

// ExecuteTemplateFunction executes a template function for this extension
//nolint:gocritic // Why: go-plugin wants this
func (s *rpcTransportServer) ExecuteTemplateFunction(t *TemplateFunctionExec, resp *[]byte) error {
	v, err := s.impl.ExecuteTemplateFunction(t)
	s.log.WithField("data", spew.Sdump(v)).WithField("name", t.Name).Debug("Extension function called")
	*resp = v
	return err
}
