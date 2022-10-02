// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the code for interacting with snapshots

package stenciltest

import (
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/pkg/stenciltest/config"
	"github.com/pkg/errors"
)

// snapshotOptions are options for Snapshot.
type snapshotOptions struct {
	// Save will save the snapshot to disk.
	Save bool

	// Persist denotes if we should save a snapshot or not, Save
	// must also be false for this to work.
	//
	// Defaults to true.
	Persist *bool

	// Validators to run on this file, outside of the validators
	// inherited from the config.
	Validators []config.Validator
}

// snapshot snapshots a codegen.File and stores it on disk, erroring
// if it differs from an already existing snapshot.
func snapshot(t *testing.T, f *codegen.File, opts *snapshotOptions) error {
	// Default persist to true if not set. When persist is true,
	// run the snapshot test, otherwise skip it.
	if opts.Persist == nil || *opts.Persist {
		snapshot := cupaloy.New(
			cupaloy.ShouldUpdate(func() bool { return opts.Save }),
			cupaloy.CreateNewAutomatically(true),
		)
		if err := snapshot.Snapshot(f); err != nil {
			return err
		}
	}

	cfg := config.GetConfig()

	// Combine the validators from the config with the validators passed in.
	//
	// Note: We do two different appends here to avoid modifying the config
	// validators slice (append modifies the slice in place sometimes).
	validators := make([]config.Validator, 0, len(cfg.Validators)+len(opts.Validators))
	validators = append(validators, cfg.Validators...)
	validators = append(validators, opts.Validators...)

	for _, v := range validators {
		if err := v.Validate(t, f.Name(), f.String()); err != nil {
			return errors.Wrap(err, "failed to validate file")
		}
	}

	return nil
}
