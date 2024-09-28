// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the stencil function passed to templates
package codegen

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/stencil/internal/jsonschema"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/go-git/go-billy/v5/util"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewStencil creates a new, fully initialized Stencil renderer function
func NewStencil(m *configuration.ServiceManifest, mods []*modules.Module, log logrus.FieldLogger) *Stencil {
	return &Stencil{
		log:         log,
		m:           m,
		ext:         extensions.NewHost(log),
		modules:     mods,
		isFirstPass: true,
		sharedData:  newSharedData(),
	}
}

// Stencil provides the basic functions for
// stencil templates
type Stencil struct {
	log logrus.FieldLogger
	m   *configuration.ServiceManifest

	ext       *extensions.Host
	extCaller *extensions.ExtensionCaller

	// modules is a list of modules used in this stencil render
	modules []*modules.Module

	// isFirstPass denotes if the renderer is currently in first
	// pass mode
	isFirstPass bool

	// sharedData is the store for module hook data and globals
	sharedData *sharedData
}

// hashModuleHookValue hashes the module hook value using the
// hashstructure library. If the hashing fails, it returns 0.
func hashModuleHookValue(m any) uint64 {
	hash, err := hashstructure.Hash(m, hashstructure.FormatV2, nil)
	if err != nil {
		hash = 0
	}
	return hash
}

// moduleHook is a wrapper type for module hook values that
// contains the values for module hooks
type moduleHook struct {
	// values are the values available for this module hook
	values []any
}

// Sort sorts the module hook values by their hash
func (m *moduleHook) Sort() {
	sort.Slice(m.values, func(i, j int) bool {
		return hashModuleHookValue(m.values[i]) < hashModuleHookValue(m.values[j])
	})
}

// global is an explicit type used to define global variables in the sharedData
// type (specifically the globals struct field) so that we can track not only the
// value of the global but also the template it came from.
type global struct {
	// template is the template that defined this global (and is scoped too)
	template string

	// value is the underlying value
	value any
}

// sharedData stores data that is injected by templates from modules
// for both module hooks and template module globals.
type sharedData struct {
	moduleHooks map[string]*moduleHook
	globals     map[string]global
}

// newSharedData returns an initialized (empty underlying maps) sharedData type.
func newSharedData() *sharedData {
	return &sharedData{
		moduleHooks: make(map[string]*moduleHook),
		globals:     make(map[string]global),
	}
}

// key returns the key name for data stored in both modulesHooks and globals.
//
// The module parameter should just be the name of the module. Key is the actual
// key passed as the identifier for either the module hook or the global.
func (*sharedData) key(module, key string) string {
	return path.Join(module, key)
}

// RegisterExtensions registers all extensions on the currently loaded
// modules.
func (s *Stencil) RegisterExtensions(ctx context.Context) error {
	for _, m := range s.modules {
		if err := m.RegisterExtensions(ctx, s.log, s.ext); err != nil {
			return errors.Wrapf(err, "failed to load extensions from module %q", m.Name)
		}
	}

	return nil
}

// RegisterInprocExtensions registers the input ext extension directly. This API is used in
// unit tests to render modules with templates that invoke native extensions: input 'ext' can be
// either an actual extension or a mock one (feeding fake data into the template).
func (s *Stencil) RegisterInprocExtensions(name string, ext apiv1.Implementation) {
	s.ext.RegisterInprocExtension(name, ext)
}

// GenerateJSONSchema generates a jsonschema.Schema using the loaded
// Stencil modules.
func (s *Stencil) GenerateJSONSchema(ctx context.Context) *jsonschema.Schema {
	schema := jsonschema.NewSchema("Service Manifest", "Service-specific schema for Stencil service manifest")
	for _, m := range s.modules {
		log := s.log.WithField("module", m.Name)
		manifest, err := m.Manifest(ctx)
		if err != nil {
			log.WithError(err).Error("Could not load module manifest")
			continue
		}

		for name, arg := range manifest.Arguments {
			log = log.WithField("name", name)
			log.WithField("arg.Schema", arg.Schema).Infof("Argument")
			if arg.From != "" {
				// This is defined in a different module
				continue
			}
			aType, ok := arg.Schema["type"].(string)
			if !ok {
				// Assume it's an array
				// FIXME: this block doesn't actually work, it always skips.
				var types []string
				types, ok = arg.Schema["type"].([]string)
				if !ok {
					log.Error("Skipping arg without type")
					continue
				}
				schema.AddProperty(name, arg.Description, jsonschema.NewTypes(types...))
			}
			switch aType {
			case "array":
				var items map[string]interface{}
				items, ok = arg.Schema["items"].(map[string]interface{})
				if !ok {
					log.Error("Skipping array arg without items")
					continue
				}
				itemType := items["type"].(string)
				schema.AddArrayProperty(name, arg.Description, itemType)
			case "object":
				// TODO: add support
			default:
				schema.AddProperty(name, arg.Description, jsonschema.NewTypes(aType))
			}
		}
	}

	return schema
}

// GenerateLockfile generates a stencil.Lockfile based
// on a list of templates.
func (s *Stencil) GenerateLockfile(tpls []*Template) *stencil.Lockfile {
	l := &stencil.Lockfile{
		Version: app.Info().Version,
	}

	for _, tpl := range tpls {
		for _, f := range tpl.Files {
			// Don't write files we skipped, or deleted, to the lockfile
			if f.Skipped || f.Deleted {
				continue
			}

			l.Files = append(l.Files, &stencil.LockfileFileEntry{
				Name:     f.Name(),
				Template: tpl.Path,
				Module:   tpl.Module.Name,
			})
		}
	}

	for _, m := range s.modules {
		l.Modules = append(l.Modules, &stencil.LockfileModuleEntry{
			Name:    m.Name,
			URL:     m.URI,
			Version: m.Version,
		})
	}

	// sort based on name to ensure deterministic output
	sort.SliceStable(l.Files, func(i, j int) bool {
		return l.Files[i].Name < l.Files[j].Name
	})

	sort.SliceStable(l.Modules, func(i, j int) bool {
		return l.Modules[i].Name < l.Modules[j].Name
	})

	return l
}

// sortModuleHooks sorts the module hooks by their hash
func (s *Stencil) sortModuleHooks() {
	for _, m := range s.sharedData.moduleHooks {
		m.Sort()
	}
}

// Render renders all templates using the ServiceManifest that was
// provided to stencil at creation time, returned is the templates
// that were produced and their associated files.
func (s *Stencil) Render(ctx context.Context, log logrus.FieldLogger) ([]*Template, error) {
	tplfiles, err := s.getTemplates(ctx, log)
	if err != nil {
		return nil, err
	}

	if s.extCaller, err = s.ext.GetExtensionCaller(ctx); err != nil {
		return nil, err
	}

	log.Debug("Creating values for template")
	vals := NewValues(ctx, s.m, s.modules)
	log.Debug("Finished creating values")

	// Add the templates to their modules template to allow them to be able to access
	// functions declared in the same module
	for _, t := range tplfiles {
		log.Debugf("Parsing template %s", t.ImportPath())
		if err := t.Parse(s); err != nil {
			return nil, errors.Wrapf(err, "failed to parse template %q", t.ImportPath())
		}
	}

	// Render the first pass, this is used to populate shared data
	for _, t := range tplfiles {
		log.Debugf("First pass render of template %s", t.ImportPath())
		if err := t.Render(s, vals); err != nil {
			return nil, errors.Wrapf(err, "failed to render template %q", t.ImportPath())
		}

		// Remove the files, we're just using this to populate the shared data.
		t.Files = nil
	}
	s.isFirstPass = false

	// Sort module hook data before the next pass
	s.sortModuleHooks()

	tpls := make([]*Template, 0)
	for _, t := range tplfiles {
		log.Debugf("Second pass render of template %s", t.ImportPath())
		if err := t.Render(s, vals); err != nil {
			return nil, errors.Wrapf(err, "failed to render template %q", t.ImportPath())
		}

		// append the rendered template to our list of templates processed
		tpls = append(tpls, t)
	}

	return tpls, nil
}

// PostRun runs all post run commands specified in the modules that
// this service depends on
func (s *Stencil) PostRun(ctx context.Context, log logrus.FieldLogger) error {
	log.Info("Running post-run command(s)")
	for _, m := range s.modules {
		mf, err := m.Manifest(ctx)
		if err != nil {
			return err
		}

		for _, cmdStr := range mf.PostRunCommand {
			log.Infof(" - %s", cmdStr.Name)
			//nolint:gosec // Why: This is by design
			cmd := exec.CommandContext(ctx, "/usr/bin/env", "bash", "-c", cmdStr.Command)
			cmd.Stdin = os.Stdin
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				return errors.Wrapf(err, "failed to run post run command for module %q", m.Name)
			}
		}
	}

	return nil
}

// getTemplates takes all modules attached to this stencil
// struct and returns all templates exposed by it.
func (s *Stencil) getTemplates(ctx context.Context, log logrus.FieldLogger) ([]*Template, error) {
	tpls := make([]*Template, 0)
	for _, m := range s.modules {
		log.Debugf("Fetching module %q", m.Name)
		fs, err := m.GetFS(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read module filesystem %q", m.Name)
		}

		// Note: This error should never really fire since we already fetched the FS above
		// that being said, we handle it here. Skip native extensions as they cannot have templates.
		mf, err := m.Manifest(ctx)
		if err != nil {
			return nil, err
		}
		if !mf.Type.Contains(configuration.TemplateRepositoryTypeTemplates) {
			log.Debugf("Skipping template discovery for module %q, not a template module (type %s)", m.Name, mf.Type)
			continue
		}

		log.Debugf("Discovering templates from module %q", m.Name)

		// default to templates/, but if it's not present fallback to
		// the root w/ a warning
		// Note: This behaviour is deprecated and will be removed soon. Put templates
		// into /templates instead.
		if inf, err := fs.Stat("templates"); err != nil || !inf.IsDir() {
			log.Warnf("Module %q has templates outside of templates/ directory, this is not recommended and is deprecated", m.Name)
		} else {
			var err error
			fs, err = fs.Chroot("templates")
			if err != nil {
				return nil, errors.Wrap(err, "failed to chroot module filesystem to templates/")
			}
		}

		err = util.Walk(fs, "", func(path string, inf os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip files without a .tpl extension
			if filepath.Ext(path) != ".tpl" {
				return nil
			}

			f, err := fs.Open(path)
			if err != nil {
				return errors.Wrapf(err, "failed to open template %q from module %q", path, m.Name)
			}
			defer f.Close()

			tplContents, err := io.ReadAll(f)
			if err != nil {
				return errors.Wrapf(err, "failed to read template %q from module %q", path, m.Name)
			}

			log.Debugf("Discovered template %q", path)
			tpl, err := NewTemplate(m, path, inf.Mode(), inf.ModTime(), tplContents, log)
			if err != nil {
				return errors.Wrapf(err, "failed to create template %q from module %q", path, m.Name)
			}
			tpls = append(tpls, tpl)

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	log.Debug("Finished discovering templates")

	return tpls, nil
}

// Close closes all resources that should be closed when done
// rendering templates.
func (s *Stencil) Close() error {
	return errors.Wrap(s.ext.Close(), "failed to close native extensions")
}
