// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements static validation of a Stencil template repository
// manifest (manifest.yaml) without resolving dependencies or accessing the
// network.

// Package manifest implements static validation of a Stencil template
// repository manifest (manifest.yaml) without resolving dependencies or
// accessing the network.
package manifest

import (
	"bytes"
	"errors"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/getoutreach/stencil/pkg/configuration"
)

// Load reads a manifest from r and decodes it twice: once strictly
// (rejecting unknown keys) to surface structural problems, and once leniently
// so the returned manifest is populated for field-level checks even when the
// strict decode failed.
//
// strictErr is the strict-decode error (if any); for empty input it is io.EOF.
// mf is nil only when the lenient decode also fails (truly malformed YAML).
// multiDoc is true when r contains more than one YAML document. readErr is
// non-nil only when reading from r itself fails.
func Load(r io.Reader) (mf *configuration.TemplateRepositoryManifest,
	strictErr error, multiDoc bool, readErr error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, false, err
	}

	// Strict decode into a throwaway manifest to capture unknown-key/type errors.
	strictDec := yaml.NewDecoder(bytes.NewReader(raw))
	strictDec.KnownFields(true)
	var strictManifest configuration.TemplateRepositoryManifest
	strictErr = strictDec.Decode(&strictManifest)

	// Lenient decode so field checks can run regardless of strictErr. Retain the
	// decoder so we can probe for a second document afterward.
	lenientDec := yaml.NewDecoder(bytes.NewReader(raw))
	var lenientManifest configuration.TemplateRepositoryManifest
	if err := lenientDec.Decode(&lenientManifest); err != nil {
		// Empty or malformed input: no usable manifest for field checks.
		return nil, strictErr, false, nil
	}
	mf = &lenientManifest

	// Probe for a second document on the (already-advanced) lenient decoder.
	var discard any
	if err := lenientDec.Decode(&discard); err == nil {
		multiDoc = true
	} else if !errors.Is(err, io.EOF) {
		// A decode error on the second document is not fatal to linting doc 1;
		// ignore it (doc 1 is what we validate).
		multiDoc = false
	}

	return mf, strictErr, multiDoc, nil
}
