// Copyright 2023 Outreach Corporation. All Rights Reserved.

// Description: This file implements a cache for jet.

package jet

import (
	"sync"

	"github.com/CloudyKit/jet/v6"
)

// cache is an in-memory cache for usage with jet. It implements the
// jet.Cache interface
type cache struct {
	m sync.Map
}

// Get retrieves a jet.Template from the cache
func (c *cache) Get(templatePath string) *jet.Template {
	_t, ok := c.m.Load(templatePath)
	if !ok {
		return nil
	}
	return _t.(*jet.Template)
}

// Put puts a template into the cache
func (c *cache) Put(templatePath string, t *jet.Template) {
	c.m.Store(templatePath, t)
}
