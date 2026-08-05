// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Pragmatic JSON Schema refinement (subschema) check used by the
// project-manifest linter to verify a module's refines: assertion. It compares
// only the schema constructs stencil uses; unhandled parent keywords fall back
// to an equality check, so a loosening is never reported as a valid narrowing.

package projectmanifest

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
)

// metadataKeywords do not constrain values and are ignored when comparing a
// child schema against its parent.
//
//nolint:gochecknoglobals // Why: static set of metadata keywords ignored during refinement.
var metadataKeywords = map[string]struct{}{
	"description": {}, "title": {}, "examples": {}, "default": {}, "$comment": {},
}

// handledKeywords are the keywords refines() compares with a dedicated rule.
// Any parent keyword outside this set (and outside metadataKeywords) falls back
// to an equality check.
//
//nolint:gochecknoglobals // Why: static set of keywords with a dedicated refinement rule.
var handledKeywords = map[string]struct{}{
	"type": {}, "enum": {}, "const": {},
	"properties": {}, "required": {}, "additionalProperties": {},
	"items": {}, "anyOf": {}, "oneOf": {},
	"minimum": {}, "maximum": {}, "minLength": {}, "maxLength": {},
	"minItems": {}, "maxItems": {}, "multipleOf": {},
	"pattern": {}, "format": {},
}

// refines reports whether child is a valid narrowing (subschema) of parent:
// every value accepted by child is also accepted by parent. On false it returns
// a human-readable reason naming the first violating keyword. Keywords are
// visited in a deterministic order so the reason is stable.
func refines(child, parent map[string]any) (ok bool, reason string) {
	// anyOf/oneOf restructure the whole comparison, so handle them first.
	if hasBranches(child) || hasBranches(parent) {
		return branchRefines(child, parent)
	}
	if ok, reason := typeRefines(child, parent); !ok {
		return false, reason
	}
	if ok, reason := enumRefines(child, parent); !ok {
		return false, reason
	}
	if ok, reason := propertiesRefines(child, parent); !ok {
		return false, reason
	}
	if ok, reason := requiredRefines(child, parent); !ok {
		return false, reason
	}
	if ok, reason := additionalPropertiesRefines(child, parent); !ok {
		return false, reason
	}
	if ok, reason := itemsRefines(child, parent); !ok {
		return false, reason
	}
	if ok, reason := boundsRefine(child, parent); !ok {
		return false, reason
	}
	if ok, reason := patternFormatRefines(child, parent); !ok {
		return false, reason
	}
	if ok, reason := fallbackRefines(child, parent); !ok {
		return false, reason
	}
	return true, ""
}

// typeRefines checks the "type" keyword: when the parent sets a type, the child
// must set a type that is a subset of it.
func typeRefines(child, parent map[string]any) (ok bool, reason string) {
	// type: parent present -> child must set type and be a subset.
	if pt, ok := parent["type"]; ok {
		ct, ok := child["type"]
		if !ok {
			return false, `child must set "type" to narrow parent's "type"`
		}
		if ok, reason := typeSubset(ct, pt); !ok {
			return false, reason
		}
	}
	return true, ""
}

// enumRefines checks enum/const: when the parent constrains its allowed values,
// the child must restrict them to a subset.
func enumRefines(child, parent map[string]any) (ok bool, reason string) {
	// enum/const: parent constrains values -> child must restrict to a subset.
	if pe := effectiveEnum(parent); pe != nil {
		ce := effectiveEnum(child)
		if ce == nil {
			return false, `child must set "enum"/"const" to narrow parent's allowed values`
		}
		if !enumSubset(ce, pe) {
			return false, `child "enum"/"const" is not a subset of parent's allowed values`
		}
	}
	return true, ""
}

// propertiesRefines checks object properties: every property the parent declares
// must be defined and refined by the child.
func propertiesRefines(child, parent map[string]any) (ok bool, reason string) {
	// object properties: every parent property must be defined and refined by child.
	if pp, ok := parent["properties"].(map[string]any); ok {
		cp, _ := child["properties"].(map[string]any)
		for _, k := range sortedKeys(pp) {
			pcs, _ := pp[k].(map[string]any)
			ccs, ok := cp[k].(map[string]any)
			if !ok {
				return false, fmt.Sprintf("child does not define property %q declared by parent", k)
			}
			if ok, reason := refines(ccs, pcs); !ok {
				return false, fmt.Sprintf("property %q: %s", k, reason)
			}
		}
	}
	return true, ""
}

// requiredRefines checks required: the child must require at least everything the
// parent requires.
func requiredRefines(child, parent map[string]any) (ok bool, reason string) {
	// required: child must require at least everything parent requires.
	if pr := toStringSet(parent["required"]); len(pr) > 0 {
		cr := toStringSet(child["required"])
		for _, k := range sortedStringSet(pr) {
			if _, ok := cr[k]; !ok {
				return false, fmt.Sprintf("child does not require property %q that parent requires", k)
			}
		}
	}
	return true, ""
}

// additionalPropertiesRefines checks additionalProperties: the child must be at
// least as strict as the parent about extra keys.
func additionalPropertiesRefines(child, parent map[string]any) (ok bool, reason string) {
	// additionalProperties: the child must be at least as strict as the parent.
	if pap, ok := parent["additionalProperties"]; ok {
		switch p := pap.(type) {
		case bool:
			if !p {
				// Parent forbids extra keys: child must forbid them too, and must
				// not introduce properties the parent does not allow.
				childAP, ok := child["additionalProperties"]
				cb, isBool := childAP.(bool)
				if !ok || !isBool || cb {
					return false, `parent sets additionalProperties:false but child does not`
				}
				pp, _ := parent["properties"].(map[string]any)
				cp, _ := child["properties"].(map[string]any)
				for _, k := range sortedKeys(cp) {
					if _, allowed := pp[k]; !allowed {
						return false, fmt.Sprintf("parent forbids additional properties but child adds %q", k)
					}
				}
			}
			// additionalProperties:true (or absent) imposes no constraint.
		case map[string]any:
			// Parent constrains extra keys with a sub-schema; child must forbid
			// them (false) or constrain them with a refining sub-schema.
			switch c := child["additionalProperties"].(type) {
			case bool:
				if c {
					return false, `parent constrains additionalProperties with a schema but child allows any`
				}
			case map[string]any:
				if ok, reason := refines(c, p); !ok {
					return false, "additionalProperties: " + reason
				}
			default:
				return false, `child does not constrain additionalProperties as parent does`
			}
			// Any property the child names that the parent does not name is, in the
			// parent, governed by the parent's additionalProperties sub-schema; the
			// child's schema for it must refine that sub-schema.
			pp, _ := parent["properties"].(map[string]any)
			cp, _ := child["properties"].(map[string]any)
			for _, k := range sortedKeys(cp) {
				if _, named := pp[k]; named {
					continue
				}
				ccs, _ := cp[k].(map[string]any)
				if ok, reason := refines(ccs, p); !ok {
					return false, fmt.Sprintf("property %q must refine the parent's additionalProperties schema: %s", k, reason)
				}
			}
		}
	}
	return true, ""
}

// patternFormatRefines checks pattern/format: the child must keep the parent's
// constraint; if both present they must be equal (subsumption of two regexes
// cannot be proven cheaply).
func patternFormatRefines(child, parent map[string]any) (ok bool, reason string) {
	for _, kw := range []string{"pattern", "format"} {
		pv, ok := parent[kw]
		if !ok {
			continue
		}
		cv, ok := child[kw]
		if !ok {
			return false, fmt.Sprintf("child drops parent's %q constraint", kw)
		}
		if canon(cv) != canon(pv) {
			return false, fmt.Sprintf("child %q differs from parent's; subsumption cannot be proven", kw)
		}
	}
	return true, ""
}

// fallbackRefines applies an equality check to any other parent keyword, so a
// keyword refines() does not model specially is never treated as a loosening.
func fallbackRefines(child, parent map[string]any) (ok bool, reason string) {
	// equality fallback for any other parent keyword.
	for _, k := range sortedKeys(parent) {
		if _, meta := metadataKeywords[k]; meta {
			continue
		}
		if _, handled := handledKeywords[k]; handled {
			continue
		}
		if canon(child[k]) != canon(parent[k]) {
			return false, fmt.Sprintf("cannot verify refinement of keyword %q: schemas differ", k)
		}
	}
	return true, ""
}

// hasBranches reports whether a schema uses anyOf or oneOf.
func hasBranches(s map[string]any) bool {
	if _, ok := s["anyOf"]; ok {
		return true
	}
	_, ok := s["oneOf"]
	return ok
}

// branchesOf returns the schema's anyOf (or oneOf) branch list, or nil when the
// schema uses neither.
func branchesOf(s map[string]any) []map[string]any {
	raw, ok := s["anyOf"]
	if !ok {
		raw, ok = s["oneOf"]
	}
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(list))
	for _, b := range list {
		if m, ok := b.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// withoutBranchKeywords returns a shallow copy of s with anyOf/oneOf removed, so
// a plain schema can be treated as a single branch.
func withoutBranchKeywords(s map[string]any) map[string]any {
	out := make(map[string]any, len(s))
	for k, v := range s {
		if k == "anyOf" || k == "oneOf" {
			continue
		}
		out[k] = v
	}
	return out
}

// branchRefines compares child and parent when either uses anyOf/oneOf: every
// effective child branch must refine at least one effective parent branch, so
// child's value set is a subset of parent's. Each side's non-branch (sibling)
// keywords are ANDed into every branch first, so a sibling constraint beside
// anyOf is not dropped. oneOf is treated like anyOf, which is conservative for a
// subset check.
func branchRefines(child, parent map[string]any) (ok bool, reason string) {
	childBranches, ok := effectiveBranches(child)
	if !ok {
		return false, "child combines anyOf/oneOf with a conflicting sibling keyword; refinement cannot be verified"
	}
	parentBranches, ok := effectiveBranches(parent)
	if !ok {
		return false, "parent combines anyOf/oneOf with a conflicting sibling keyword; refinement cannot be verified"
	}
	for i, cb := range childBranches {
		matched := false
		for _, pb := range parentBranches {
			if ok, _ := refines(cb, pb); ok {
				matched = true
				break
			}
		}
		if !matched {
			return false, fmt.Sprintf("child branch %d does not refine any parent branch", i)
		}
	}
	return true, ""
}

// effectiveBranches expands a schema into its anyOf/oneOf branches with the
// schema's non-branch (sibling) keywords merged into each branch. A schema
// without branches is a single branch (itself). It reports ok=false when a
// sibling and a branch constrain the same keyword with different values, since a
// single map cannot represent that AND; the caller then fails conservatively.
func effectiveBranches(s map[string]any) ([]map[string]any, bool) {
	siblings := withoutBranchKeywords(s)
	branches := branchesOf(s)
	if branches == nil {
		return []map[string]any{siblings}, true
	}
	out := make([]map[string]any, 0, len(branches))
	for _, b := range branches {
		merged, ok := mergeSchemas(siblings, b)
		if !ok {
			return nil, false
		}
		out = append(out, merged)
	}
	return out, true
}

// mergeSchemas returns the AND of two schemas as a single map when they do not
// constrain the same keyword with different values, reporting ok=false on a
// conflicting keyword.
func mergeSchemas(a, b map[string]any) (map[string]any, bool) {
	out := make(map[string]any, len(a)+len(b))
	maps.Copy(out, a)
	for k, v := range b {
		if existing, ok := out[k]; ok && canon(existing) != canon(v) {
			return nil, false
		}
		out[k] = v
	}
	return out, true
}

// typeSubset reports whether child's declared type(s) are all within parent's.
func typeSubset(childType, parentType any) (ok bool, reason string) {
	cs := toStringSet(childType)
	if len(cs) == 0 {
		return false, `child "type" is not a string or list of strings`
	}
	ps := toStringSet(parentType)
	for _, t := range sortedStringSet(cs) {
		if _, ok := ps[t]; !ok {
			return false, fmt.Sprintf("type %q is not within parent type %s", t, renderType(parentType))
		}
	}
	return true, ""
}

// renderType formats a type value for an error message (quoting a bare string).
func renderType(t any) string {
	if s, ok := t.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return canon(t)
}

// effectiveEnum returns the schema's allowed-value list: its enum, or a
// single-element list from const, or nil when it constrains neither.
func effectiveEnum(s map[string]any) []any {
	if e, ok := s["enum"].([]any); ok {
		return e
	}
	if c, ok := s["const"]; ok {
		return []any{c}
	}
	return nil
}

// enumSubset reports whether every child value appears in parent's value set.
func enumSubset(child, parent []any) bool {
	pset := make(map[string]struct{}, len(parent))
	for _, v := range parent {
		pset[canon(v)] = struct{}{}
	}
	for _, v := range child {
		if _, ok := pset[canon(v)]; !ok {
			return false
		}
	}
	return true
}

// itemsRefines compares array item schemas. It supports both the single-schema
// form (items: {...}) and the tuple form (items: [{...}, ...]). When parent has
// no items constraint, child may add one.
func itemsRefines(child, parent map[string]any) (ok bool, reason string) {
	pit, ok := parent["items"]
	if !ok {
		return true, "" // parent unconstrained; child may add items
	}
	cit, ok := child["items"]
	if !ok {
		return false, `child does not constrain "items" as parent does`
	}
	if pm, ok := pit.(map[string]any); ok {
		cm, ok := cit.(map[string]any)
		if !ok {
			return false, `child "items" shape does not match parent's`
		}
		if ok, reason := refines(cm, pm); !ok {
			return false, "items: " + reason
		}
		return true, ""
	}
	if pl, ok := pit.([]any); ok {
		cl, ok := cit.([]any)
		if !ok || len(cl) != len(pl) {
			return false, `child "items" tuple does not match parent's length`
		}
		for i := range pl {
			pm, _ := pl[i].(map[string]any)
			cm, _ := cl[i].(map[string]any)
			if ok, reason := refines(cm, pm); !ok {
				return false, fmt.Sprintf("items[%d]: %s", i, reason)
			}
		}
	}
	return true, ""
}

// boundsRefine checks numeric and length bounds. Minimum-style bounds may only
// increase (or be added); maximum-style bounds may only decrease (or be added);
// multipleOf may be added or replaced by an integer multiple of parent's.
func boundsRefine(child, parent map[string]any) (ok bool, reason string) {
	for _, kw := range []string{"minimum", "minLength", "minItems"} {
		if ok, reason := boundDirection(child, parent, kw, true); !ok {
			return false, reason
		}
	}
	for _, kw := range []string{"maximum", "maxLength", "maxItems"} {
		if ok, reason := boundDirection(child, parent, kw, false); !ok {
			return false, reason
		}
	}
	if pv, ok := asFloat(parent["multipleOf"]); ok {
		cv, ok := asFloat(child["multipleOf"])
		if !ok {
			return false, `child drops parent's "multipleOf" constraint`
		}
		if pv == 0 || math.Mod(cv, pv) != 0 {
			return false, `child "multipleOf" is not an integer multiple of parent's`
		}
	}
	return true, ""
}

// boundDirection enforces one bound keyword. When wantHigher is true (minimum
// family) child must be >= parent; otherwise (maximum family) child must be <=
// parent. A child bound where parent has none is always a valid tightening.
func boundDirection(child, parent map[string]any, kw string, wantHigher bool) (ok bool, reason string) {
	pv, ok := asFloat(parent[kw])
	if !ok {
		return true, ""
	}
	cv, ok := asFloat(child[kw])
	if !ok {
		return false, fmt.Sprintf("child drops parent's %q bound", kw)
	}
	if wantHigher && cv < pv {
		return false, fmt.Sprintf("child %q %v is looser than parent's %v", kw, cv, pv)
	}
	if !wantHigher && cv > pv {
		return false, fmt.Sprintf("child %q %v is looser than parent's %v", kw, cv, pv)
	}
	return true, ""
}

// asFloat coerces a numeric schema value to float64.
func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// toStringSet coerces a string or list-of-strings schema value to a set.
func toStringSet(v any) map[string]struct{} {
	out := map[string]struct{}{}
	switch t := v.(type) {
	case string:
		out[t] = struct{}{}
	case []any:
		for _, e := range t {
			if s, ok := e.(string); ok {
				out[s] = struct{}{}
			}
		}
	case []string:
		for _, s := range t {
			out[s] = struct{}{}
		}
	}
	return out
}

// canon returns a canonical JSON encoding of v for order-insensitive equality.
func canon(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// sortedKeys returns a schema map's keys in sorted order.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// sortedStringSet returns a string set's members in sorted order.
func sortedStringSet(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
