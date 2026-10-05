// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements tests for the stencil command.

package stencil

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	return log
}

func TestCommand_useModulesFromLock(t *testing.T) {
	type fields struct {
		lock                      *stencil.Lockfile
		manifest                  *configuration.ServiceManifest
		dryRun                    bool
		allowMajorVersionUpgrades bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    []*configuration.TemplateRepository
	}{
		{
			name: "should fail if lockfile is nil",
			fields: fields{
				lock: nil,
				manifest: &configuration.ServiceManifest{
					Modules: []*configuration.TemplateRepository{
						{
							Name: "testing",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should use module version from lockfile",
			fields: fields{
				lock: &stencil.Lockfile{
					Modules: []*stencil.LockfileModuleEntry{
						{
							Name:    "testing",
							Version: "1.0.0",
						},
					},
				},
				manifest: &configuration.ServiceManifest{
					Modules: []*configuration.TemplateRepository{
						{
							Name: "testing",
						},
					},
				},
			},
			wantErr: false,
			want: []*configuration.TemplateRepository{
				{
					Name:    "testing",
					Version: "1.0.0",
				},
			},
		},
		{
			name: "should add module from lockfile if not in manifest as top-level module",
			fields: fields{
				lock: &stencil.Lockfile{
					Modules: []*stencil.LockfileModuleEntry{
						{
							Name:    "testing",
							Version: "1.0.0",
						},
					},
				},
				manifest: &configuration.ServiceManifest{
					Modules: []*configuration.TemplateRepository{},
				},
			},
			wantErr: false,
			want: []*configuration.TemplateRepository{
				{
					Name:    "testing",
					Version: "1.0.0",
				},
			},
		},
		{
			name: "should error if manifest has module not in lockfile",
			fields: fields{
				lock: &stencil.Lockfile{
					Modules: []*stencil.LockfileModuleEntry{},
				},
				manifest: &configuration.ServiceManifest{
					Modules: []*configuration.TemplateRepository{
						{
							Name: "testing",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return stable version returned by channel that's not longer selectable",
			fields: fields{
				lock: &stencil.Lockfile{
					Modules: []*stencil.LockfileModuleEntry{
						{
							Name:    "testing",
							Version: "1.0.0",
						},
					},
				},
				manifest: &configuration.ServiceManifest{
					Modules: []*configuration.TemplateRepository{
						{
							Name:    "testing",
							Channel: "rc",
						},
					},
				},
			},
			want: []*configuration.TemplateRepository{
				{
					Name:    "testing",
					Version: "1.0.0",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				lock:                      tt.fields.lock,
				manifest:                  tt.fields.manifest,
				log:                       testLogger(t),
				dryRun:                    tt.fields.dryRun,
				frozenLockfile:            true,
				allowMajorVersionUpgrades: tt.fields.allowMajorVersionUpgrades,
			}
			if err := c.useModulesFromLock(); (err != nil) != tt.wantErr {
				t.Errorf("Command.useModulesFromLock() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want != nil {
				if diff := cmp.Diff(tt.want, tt.fields.manifest.Modules); diff != "" {
					t.Errorf("Command.useModulesFromLock() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestValidateStencilVersionBadVersion(t *testing.T) {
	ctx := context.Background()
	c := &Command{
		log: testLogger(t),
	}
	mods := []*modules.Module{
		modules.NewWithFS(ctx, "example.com/stencil-test", osfs.New("testdata/stencil-version-success")),
	}
	err := c.validateStencilVersion(ctx, mods, "invalid")
	assert.ErrorIs(t, err, semver.ErrInvalidSemVer)
}

func TestValidateStencilVersionBadConstraint(t *testing.T) {
	ctx := context.Background()
	c := &Command{
		log: testLogger(t),
	}
	mods := []*modules.Module{
		modules.NewWithFS(ctx, "example.com/stencil-test", osfs.New("testdata/stencil-version-bad-constraint")),
	}
	err := c.validateStencilVersion(ctx, mods, "v1.10.0")
	assert.Error(t, err, `improper constraint: "invalid"`)
}

func TestValidateStencilVersionConstraintValidationFailure(t *testing.T) {
	ctx := context.Background()
	c := &Command{
		log: testLogger(t),
	}
	mods := []*modules.Module{
		modules.NewWithFS(ctx, "example.com/stencil-test", osfs.New("testdata/stencil-version-failure")),
	}
	err := c.validateStencilVersion(ctx, mods, "v1.10.0")
	assert.ErrorContains(t, err, "stencil version v1.10.0 does not match the version constraint (^2.0.0) for example.com/stencil-test")
}

func TestValidateStencilVersionConstraintValidationSuccess(t *testing.T) {
	ctx := context.Background()
	c := &Command{
		log: testLogger(t),
	}
	mods := []*modules.Module{
		modules.NewWithFS(ctx, "example.com/stencil-test", osfs.New("testdata/stencil-version-success")),
	}
	assert.NilError(t, c.validateStencilVersion(ctx, mods, "v1.10.0"))
}

func TestCommand_writeFileDelete(t *testing.T) {
	tests := []struct {
		name       string
		exists     bool
		skipped    bool
		dryRun     bool
		wantAction string
		wantExists bool
	}{
		{name: "should delete an existing file", exists: true, wantAction: "Deleted"},
		{name: "should delete an existing file that is also skipped", exists: true, skipped: true, wantAction: "Deleted"},
		{name: "should not report a missing file as deleted", wantAction: "Already absent"},
		{name: "should keep an existing file in dry-run", exists: true, dryRun: true, wantAction: "Deleted", wantExists: true},
		{name: "should not report a missing file as deleted in dry-run", dryRun: true, wantAction: "Already absent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "file.txt")
			if tt.exists {
				assert.NilError(t, os.WriteFile(path, []byte("contents"), 0o644))
			}
			f, err := codegen.NewFile(path, 0o644, time.Now())
			assert.NilError(t, err)
			f.Deleted = true
			f.Skipped = tt.skipped

			var out bytes.Buffer
			log := logrus.New()
			log.SetOutput(&out)
			c := &Command{log: log, dryRun: tt.dryRun}

			assert.NilError(t, c.writeFile(f))
			assert.Assert(t, strings.Contains(out.String(), "-> "+tt.wantAction+" "+path), out.String())
			_, err = os.Lstat(path)
			assert.Equal(t, err == nil, tt.wantExists)
		})
	}
}

func TestCommand_writeFileDeleteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dir")
	f, err := codegen.NewFile(path, 0o644, time.Now())
	assert.NilError(t, err)
	f.Deleted = true

	// os.Remove fails on a non-empty directory
	assert.NilError(t, os.MkdirAll(filepath.Join(path, "child"), 0o755))

	c := &Command{log: testLogger(t)}
	assert.ErrorContains(t, c.writeFile(f), "failed to delete")
}
