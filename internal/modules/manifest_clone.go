// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file implements deep copying of module manifests.

package modules

import (
	"slices"

	"github.com/getoutreach/stencil/pkg/configuration"
)

// cloneManifest returns a copy of mf that shares no maps, slices, or pointers
// with it, except for the slice behind Type, which is copied by value. Callers
// must not modify the result of Type.Types().
func cloneManifest(mf *configuration.TemplateRepositoryManifest) configuration.TemplateRepositoryManifest {
	out := *mf
	out.Modules = clonePtrs(mf.Modules)
	out.PostRunCommand = clonePtrs(mf.PostRunCommand)

	if mf.Arguments != nil {
		out.Arguments = make(map[string]configuration.Argument, len(mf.Arguments))
		for name, arg := range mf.Arguments {
			out.Arguments[name] = cloneArgument(&arg)
		}
	}

	return out
}

// cloneArgument returns a copy of arg that shares no maps or slices with it.
func cloneArgument(arg *configuration.Argument) configuration.Argument {
	out := *arg
	out.Default = cloneValue(arg.Default)
	out.Schema = cloneMap(arg.Schema)
	//nolint:staticcheck // Why: Values is deprecated but must still be copied.
	out.Values = slices.Clone(arg.Values)
	return out
}

// clonePtrs returns a copy of s in which each non-nil element points to a
// shallow copy of the original element. A nil s stays nil.
func clonePtrs[T any](s []*T) []*T {
	if s == nil {
		return nil
	}

	out := make([]*T, len(s))
	for i, p := range s {
		if p != nil {
			c := *p
			out[i] = &c
		}
	}
	return out
}

// cloneMap deep copies a map as produced by YAML decoding. A nil m stays nil.
func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneValue(v)
	}
	return out
}

// cloneValue deep copies the maps and slices that YAML decoding produces for
// an untyped value. Other values are immutable and returned as is.
func cloneValue(v any) any {
	switch v := v.(type) {
	case map[string]any:
		return cloneMap(v)
	case map[any]any:
		out := make(map[any]any, len(v))
		for k, e := range v {
			out[k] = cloneValue(e)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = cloneValue(e)
		}
		return out
	default:
		return v
	}
}
