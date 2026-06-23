// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: Tests for render-time deprecated-argument warnings.

package codegen

import (
	"context"
	"testing"

	"github.com/getoutreach/stencil/pkg/configuration"
	"gotest.tools/v3/assert"
)

func TestDeprecatedArgumentWarnings(t *testing.T) {
	t.Run("warns when a deprecated arg is set in service.yaml", func(t *testing.T) {
		tt := fakeTemplate(t,
			map[string]interface{}{"oldArg": "x"}, // service.yaml sets it
			map[string]configuration.Argument{
				"oldArg": {Deprecated: "use newArg instead"},
			})
		got, err := tt.s.deprecatedArgumentWarnings(context.Background())
		assert.NilError(t, err)
		assert.DeepEqual(t, got, []string{
			`module "test" argument "oldArg" is deprecated: use newArg instead`,
		})
	})

	t.Run("no warning when the deprecated arg is not set", func(t *testing.T) {
		tt := fakeTemplate(t,
			map[string]interface{}{}, // service.yaml does NOT set it
			map[string]configuration.Argument{
				"oldArg": {Deprecated: "use newArg instead"},
			})
		got, err := tt.s.deprecatedArgumentWarnings(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, len(got), 0)
	})

	t.Run("no warning for a non-deprecated arg that is set", func(t *testing.T) {
		tt := fakeTemplate(t,
			map[string]interface{}{"normalArg": "x"},
			map[string]configuration.Argument{
				"normalArg": {},
			})
		got, err := tt.s.deprecatedArgumentWarnings(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, len(got), 0)
	})

	t.Run("deterministic sorted order for multiple deprecated args", func(t *testing.T) {
		tt := fakeTemplate(t,
			map[string]interface{}{"bArg": "x", "aArg": "y"},
			map[string]configuration.Argument{
				"bArg": {Deprecated: "msg b"},
				"aArg": {Deprecated: "msg a"},
			})
		got, err := tt.s.deprecatedArgumentWarnings(context.Background())
		assert.NilError(t, err)
		assert.DeepEqual(t, got, []string{
			`module "test" argument "aArg" is deprecated: msg a`,
			`module "test" argument "bArg" is deprecated: msg b`,
		})
	})

	t.Run("from: re-export is skipped", func(t *testing.T) {
		// test-0 re-exports "shared" from test-1 via from:; test-1 owns the
		// deprecation. The from: entry must NOT warn.
		tt := fakeTemplateMultipleModules(t,
			map[string]interface{}{"shared": "x"},
			// test-0 (the importer)
			map[string]configuration.Argument{
				"shared": {From: "test-1", Deprecated: "ignored on from"},
			},
			// test-1 (the owner) — note: owner is imported, so it warns
			map[string]configuration.Argument{
				"shared": {Deprecated: "use the new shared"},
			},
		)
		got, err := tt.s.deprecatedArgumentWarnings(context.Background())
		assert.NilError(t, err)
		// Only the owning module (test-1) warns; the from: entry in test-0 is skipped.
		assert.DeepEqual(t, got, []string{
			`module "test-1" argument "shared" is deprecated: use the new shared`,
		})
	})
}
