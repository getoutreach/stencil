// Copyright 2021 Outreach Corporation. All Rights Reserved.

// Description: Hooks in processor framework to the builder.

package codegen

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/processors"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ProcessWalkFunc implements a filepath.WalkFunc function that runs either post or pre
// processors (depending on params passed in) on all files. If preCodegen is false, it is
// assumed to be post-codegen.
func (b *Builder) ProcessWalkFunc(preCodegen bool, log logrus.FieldLogger) filepath.WalkFunc {
	return func(fp string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			// Skip symlinks.
			return nil
		}

		f, err := os.Open(fp)
		if err != nil {
			return errors.Wrapf(err, "open file '%s'", fp)
		}
		defer f.Close()

		var processedFile *processors.File

		if preCodegen {
			processedFile, err = b.Processors.RunPreCodegen(processors.NewFile(f, fp), nil)
		} else {
			// Assume post-codegen if preCodegen is false.
			processedFile, err = b.Processors.RunPostCodegen(processors.NewFile(f, fp), nil)
		}

		if err != nil && err != processors.ErrNotProcessable {
			return errors.Wrap(err, "failed to process file")
		} else if err == processors.ErrNotProcessable {
			// Skip file.
			return nil
		}

		perms := os.FileMode(0644)
		if strings.HasSuffix(fp, ".sh") {
			perms = os.FileMode(0744)
		}

		data, err := ioutil.ReadAll(processedFile)
		if err != nil {
			return errors.Wrap(err, "read processed file")
		}

		log.Infof("Processed file '%s' in post-processing step", fp)
		if err := ioutil.WriteFile(fp, data, perms); err != nil {
			return errors.Wrap(err, "failed to write post-processed file")
		}

		return nil
	}
}

// PreCodegenProcessors runs any pre-codgegen processors.
func (b *Builder) PreCodegenProcessors(ctx context.Context, log logrus.FieldLogger) error {
	log.Info("Running pre-processors")
	if err := filepath.Walk(".", b.ProcessWalkFunc(true, log)); err != nil {
		return errors.Wrap(err, "run pre-processors on all files")
	}

	// Refetch the service manifest if it's changed.
	svcManifest, err := configuration.NewDefaultServiceManifest()
	if err != nil {
		return errors.Wrap(err, "read service manifest")
	}
	b.Manifest = svcManifest

	return nil
}

// PostCodegenProcessors runs any post-codgegen processors and reruns codegen if any of them
// require that.
func (b *Builder) PostCodegenProcessors(ctx context.Context, log logrus.FieldLogger) error {
	log.Info("Running post-processors")
	if err := filepath.Walk(".", b.ProcessWalkFunc(false, log)); err != nil {
		return errors.Wrap(err, "run post-processors on all files")
	}

	if b.Processors.ShouldRerunPostCodegen() {
		warnings, err := b.Run(ctx)
		for _, warning := range warnings {
			log.Warn(warning)
		}

		if err != nil {
			return errors.Wrap(err, "rerun codegen after processors")
		}
	}

	return nil
}
