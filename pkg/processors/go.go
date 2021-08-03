package processors

import (
	"bytes"
	"io"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

type GoMod struct{}

func (g *GoMod) Register() *Config {
	return &Config{
		FileNames: []string{"go.mod"},
	}
}

func (g *GoMod) Process(orig, template *File) (*File, error) { //nolint:funlen
	// If we didn't have an original file, then just return the template.
	if orig == nil {
		return template, nil
	}

	origModBytes, err := io.ReadAll(orig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing go.mod")
	}

	origMod, err := modfile.Parse("go.mod", origModBytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse existing go.mod")
	}

	originalModHM := make(map[string]semver.Version)
	for _, mod := range origMod.Require {
		//nolint:govet // Why: We're OK shadowing err
		v, err := semver.ParseTolerant(mod.Mod.Version)
		if err != nil {
			continue
		}

		originalModHM[mod.Mod.Path] = v
	}

	templateModbytes, err := io.ReadAll(template)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read generated go.mod")
	}

	templateMod, err := modfile.Parse("go.generated.mod", templateModbytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse generated go.mod")
	}

	for _, req := range templateMod.Require {
		//nolint:govet /// Why: We're OK shadowing err
		v, err := semver.ParseTolerant(req.Mod.Version)
		if err != nil {
			continue
		}

		// If it already exists, skip it if it's newer or equal to the one we want
		if origVer, ok := originalModHM[req.Mod.Path]; ok && origVer.GTE(v) {
			continue
		}

		err = origMod.AddRequire(req.Mod.Path, req.Mod.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add/update dependency '%s'", req.Mod.Path)
		}
	}

	for _, repl := range templateMod.Replace {
		// This isn't great performance, but I suspect nobody will have a large
		// enough amount of replaces that this will ever matter. If it does,
		// I'm sorry :(
		alreadyFound := false
		for _, origRepl := range origMod.Replace {
			// Check if we have a replace that matches
			if origRepl.New.Path == repl.New.Path &&
				origRepl.Old.Path == repl.Old.Path {
				alreadyFound = true
				break
			}
		}

		if alreadyFound {
			break
		}

		err = origMod.AddReplace(repl.Old.Path, repl.Old.Version, repl.New.Path, repl.New.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add replace: %v", repl)
		}
	}

	err = origMod.AddGoStmt(templateMod.Go.Version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set go version")
	}

	newBytes, err := origMod.Format()
	if err != nil {
		return nil, errors.Wrap(err, "failed to save generated go.mod")
	}

	return &File{bytes.NewBuffer(newBytes), "go.mod"}, err
}
