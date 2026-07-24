// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements safe, comment-preserving auto-fixes for a Stencil
// project manifest (service.yaml). The only migration is the shared module
// prerelease -> channel fix; service.yaml has no per-argument schema to migrate.

package projectmanifest

import (
	"bytes"

	"go.yaml.in/yaml/v3"

	"github.com/getoutreach/stencil/internal/lint/modulefix"
)

// Applied records one fix the fixer made, for logging. Aliased to the shared
// modulefix.Applied so the command layer can log manifest and project-manifest
// fixes uniformly.
type Applied = modulefix.Applied

// FixBytes decodes raw, applies the shared module-prerelease migration, and
// re-encodes at 2-space indent. ok is false only when raw cannot be decoded as
// YAML, in which case the caller should skip fixing and run the normal lint
// (which reports the decode error). A valid document with no modules: sequence
// (or nothing to fix) is a no-op that returns raw verbatim with ok true, so a
// no-op --fix never reformats the file. Mirrors manifest.FixBytes.
func FixBytes(raw []byte) (fixed []byte, applied []Applied, ok bool) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, false
	}
	if len(doc.Content) == 0 {
		return raw, nil, true
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return raw, nil, true
	}
	applied = modulefix.FixModules(root)
	if len(applied) == 0 {
		return raw, nil, true
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, nil, false
	}
	if err := enc.Close(); err != nil {
		return nil, nil, false
	}
	return buf.Bytes(), applied, true
}
