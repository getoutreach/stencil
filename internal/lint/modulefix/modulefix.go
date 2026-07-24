// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Shared, comment-preserving auto-fix for deprecated module fields
// in a Stencil manifest's or project's modules: list. Both service.yaml and
// manifest.yaml use the same module DOM (name / prerelease / channel), so this
// migration is shared.

// Package modulefix implements the shared module-deprecation auto-fix used by
// both the manifest and project-manifest linters.
package modulefix

import (
	"strconv"

	"go.yaml.in/yaml/v3"

	"github.com/getoutreach/stencil/internal/lint/yamlfix"
)

// Applied records one fix the fixer made, for logging. Path mirrors the
// corresponding lint.Finding.Path (e.g. "modules.<name>.prerelease").
type Applied struct {
	Path    string
	Message string
}

// FixModules walks the modules: sequence in root (a manifest/service root
// mapping node) and applies the safe module-field migrations in place, returning
// what it changed. A missing or non-sequence modules: key is a no-op.
func FixModules(root *yaml.Node) []Applied {
	var applied []Applied
	mi := yamlfix.FindKey(root, "modules")
	if mi < 0 {
		return applied
	}
	modules := root.Content[mi+1]
	if modules.Kind != yaml.SequenceNode {
		return applied
	}
	for i, raw := range modules.Content {
		mod := yamlfix.Deref(raw)
		if mod.Kind != yaml.MappingNode {
			continue
		}
		fixModulePrerelease(modulePathFor(mod, i), mod, &applied)
	}
	return applied
}

// modulePathFor builds the finding path for module i, preferring its name.
func modulePathFor(mod *yaml.Node, i int) string {
	name := ""
	if ni := yamlfix.FindKey(mod, "name"); ni >= 0 {
		name = mod.Content[ni+1].Value
	}
	return ModulePath(name, i)
}

// ModulePath builds the finding path for module i, preferring its name over its
// slice index. Shared so the checker and fixer produce identical paths.
func ModulePath(name string, i int) string {
	if name != "" {
		return "modules." + name
	}
	return "modules[" + strconv.Itoa(i) + "]"
}

// fixModulePrerelease migrates a deprecated module prerelease field. A true
// value becomes channel: rc, unless channel is already set, in which case
// channel is preserved and prerelease is just dropped; a false value is simply
// removed as a redundant default.
func fixModulePrerelease(modPath string, mod *yaml.Node, applied *[]Applied) {
	i := yamlfix.FindKey(mod, "prerelease")
	if i < 0 {
		return
	}
	val := mod.Content[i+1]
	if val.Kind != yaml.ScalarNode {
		return
	}
	switch val.Value {
	case "true":
		srcKey := mod.Content[i] // capture before RemoveKey
		yamlfix.RemoveKey(mod, "prerelease")
		if yamlfix.SetIfAbsent(mod, "channel",
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "rc"}) {
			yamlfix.CarryKeyComments(mod, "channel", srcKey)
			*applied = append(*applied, Applied{
				Path:    modPath + ".prerelease",
				Message: "migrated 'prerelease: true' to 'channel: rc'",
			})
		} else {
			*applied = append(*applied, Applied{
				Path:    modPath + ".prerelease",
				Message: "removed deprecated 'prerelease' (channel already set)",
			})
		}
	case "false":
		yamlfix.RemoveKey(mod, "prerelease")
		*applied = append(*applied, Applied{
			Path:    modPath + ".prerelease",
			Message: "removed redundant 'prerelease: false'",
		})
	}
}
