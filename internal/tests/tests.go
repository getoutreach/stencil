package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/ulikunitz/xz"
)

type Test struct {
	Name            string `json:"name"`
	ManifestEncoded string `json:"manifest,omitempty"`
}

// Manifest parses the underlying encoded manifest and returns it
func (t *Test) Manifest() (*configuration.ServiceManifest, error) {
	xzr, err := xz.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(t.ManifestEncoded)))
	if err != nil {
		return nil, err
	}

	var m *configuration.ServiceManifest
	return m, json.NewDecoder(xzr).Decode(&m) //nolint:gocritic // Why: We're OK w/ the eval order
}

// NewTest creates a new test that should be sent over the wire.
func NewTest(name string, m *configuration.ServiceManifest) (*Test, error) {
	buf := new(bytes.Buffer)
	xzwr, err := xz.NewWriter(buf)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(xzwr).Encode(m)
	if err != nil {
		return nil, err
	}
	err = xzwr.Close()
	if err != nil {
		return nil, err
	}

	return &Test{name, base64.StdEncoding.EncodeToString(buf.Bytes())}, nil
}
