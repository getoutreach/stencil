// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file tests that cloneManifest handles every field.

package modules

import (
	"reflect"
	"slices"
	"testing"

	"github.com/getoutreach/stencil/pkg/configuration"
	"gotest.tools/v3/assert"
)

// TestCloneManifestHandlesEveryReferenceField fails when a copied type gains a
// map, slice, pointer, interface, or struct field that is not listed here. The
// test checks only the list, not the copy. After adding a field, copy it in
// cloneManifest or cloneArgument, then list it here.
func TestCloneManifestHandlesEveryReferenceField(t *testing.T) {
	handled := map[reflect.Type]map[string]bool{
		reflect.TypeFor[configuration.TemplateRepositoryManifest](): {
			"Modules": true, "Type": true, "PostRunCommand": true, "Arguments": true,
		},
		reflect.TypeFor[configuration.Argument](): {
			"Default": true, "Schema": true, "Values": true,
		},
		// These have only scalar fields, so a shallow copy is a full copy.
		reflect.TypeFor[configuration.TemplateRepository](): {},
		reflect.TypeFor[configuration.PostRunCommandSpec](): {},
	}

	referenceKinds := []reflect.Kind{reflect.Map, reflect.Slice, reflect.Pointer, reflect.Interface, reflect.Struct}
	for typ, fields := range handled {
		for i := range typ.NumField() {
			field := typ.Field(i)
			if slices.Contains(referenceKinds, field.Type.Kind()) {
				assert.Assert(t, fields[field.Name],
					"%s.%s is a reference type that cloneManifest may not copy", typ.Name(), field.Name)
			}
		}
	}
}
