// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description.

// Package extensions consumes extensions in stencil
package extensions

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/getoutreach/gobox/pkg/github"
	"github.com/getoutreach/gobox/pkg/updater"
	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	giturls "github.com/whilp/git-urls"

	gogithub "github.com/google/go-github/v43/github"
)

// Host implements an extension host that handles
// registering extensions and executing them.
type Host struct {
	extensions map[string]apiv1.Implementation
}

// NewHost creates a new extension host
func NewHost() *Host {
	return &Host{
		extensions: make(map[string]apiv1.Implementation),
	}
}

// createFunctionFromTemplateFunction takes a given
// TemplateFunction and turns it into a callable function
func (h *Host) createFunctionFromTemplateFunction(extName string, ext apiv1.Implementation, fn *apiv1.TemplateFunction) reflect.Value {
	extPath := extName + "." + fn.Name

	// convert the arguments from an interface into their reflect.Types
	argTypes := make([]reflect.Type, len(fn.ArgumentTypes))
	argTypesStr := make([]string, len(argTypes))
	for i, v := range fn.ArgumentTypes {
		argTypes[i] = reflect.TypeOf(v)
		argTypesStr[i] = argTypes[i].String()
	}

	// create the return signature of <type>, error
	returnTypes := []reflect.Type{reflect.TypeOf(fn.ReturnType), reflect.TypeOf((*error)(nil)).Elem()}
	returnTypesStr := make([]string, len(returnTypes))
	for i, v := range returnTypes {
		returnTypesStr[i] = v.String()
	}

	// signature: func(<args>) (interface{}, error)
	generatedFnType := reflect.FuncOf(argTypes, returnTypes, false)
	return reflect.MakeFunc(generatedFnType, func(reflectArgs []reflect.Value) []reflect.Value {
		args := make([]interface{}, len(reflectArgs))
		for i, v := range reflectArgs {
			args[i] = v.Interface()
		}

		resp, err := ext.ExecuteTemplateFunction(&apiv1.TemplateFunctionExec{
			Name:      fn.Name,
			Arguments: args,
		})

		// returning nil is a zero value, so create a zero value of the intended value
		respRetr := reflect.ValueOf(resp)
		if resp == nil {
			respRetr = reflect.New(reflect.TypeOf(fn.ReturnType)).Elem()
		}

		// ensure that we're returning the correct type of zero
		// value for error if it's nil
		var errRetr reflect.Value
		if err == nil {
			errRetr = reflect.New(reflect.TypeOf((*error)(nil)).Elem()).Elem()
		} else {
			// wrap the error with a friend error message
			errRetr = reflect.ValueOf(errors.Wrapf(err, "failed to run extension '%s'", extPath))
		}

		// ensure that we don't return a zero value when just an error is returned
		return []reflect.Value{respRetr, errRetr}
	})
}

// GetExtensionCaller returns an extension caller that's
// aware of all extension functions
func (h *Host) GetExtensionCaller(ctx context.Context) (*ExtensionCaller, error) {
	// funcMap stores the extension functions discovered
	funcMap := map[string]map[string]reflect.Value{}

	// Call all extensions to get the template functions provided
	for extName, ext := range h.extensions {
		funcs, err := ext.GetTemplateFunctions()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get template functions from plugin '%s'", extName)
		}

		for _, f := range funcs {
			tfunc := h.createFunctionFromTemplateFunction(extName, ext, f)

			if _, ok := funcMap[extName]; !ok {
				funcMap[extName] = make(map[string]reflect.Value)
			}
			funcMap[extName][f.Name] = tfunc
		}
	}

	// return the lookup function, used via Call()
	return &ExtensionCaller{funcMap}, nil
}

// RegisterExtension registers a ext from a given source
// and compiles/downloads it. A client is then created
// that is able to communicate with the ext.
func (h *Host) RegisterExtension(ctx context.Context, source, name, version string) error { //nolint:funlen // Why: OK length.
	u, err := giturls.Parse(source)
	if err != nil {
		return errors.Wrap(err, "failed to parse extension URL")
	}

	var extPath string
	if u.Scheme == "file" {
		extPath, err = h.buildFromLocal(ctx, filepath.Join(u.Path, name), name)
	} else {
		pathSpl := strings.Split(u.Path, "/")
		if len(pathSpl) < 2 {
			return fmt.Errorf("invalid repository, expected org/repo, got %s", u.Path)
		}
		extPath, err = h.downloadFromRemote(ctx, pathSpl[0], pathSpl[1], name, version)
	}
	if err != nil {
		return errors.Wrap(err, "failed to setup extension")
	}

	// create a connection to the extension
	client := plugin.NewClient(&plugin.ClientConfig{
		Logger: hclog.New(&hclog.LoggerOptions{
			Level:  hclog.Info,
			Output: os.Stderr,
		}),
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  apiv1.Version,
			MagicCookieKey:   apiv1.CookieKey,
			MagicCookieValue: apiv1.CookieValue,
		},
		Plugins: map[string]plugin.Plugin{
			apiv1.Name: &apiv1.ExtensionPlugin{},
		},
		Cmd: exec.CommandContext(ctx, extPath),
	})

	rpcClient, err := client.Client()
	if err != nil {
		return errors.Wrap(err, "failed to create connection to extension")
	}

	raw, err := rpcClient.Dispense(apiv1.Name)
	if err != nil {
		return errors.Wrap(err, "failed to setup extension connection over extension")
	}

	ext, ok := raw.(apiv1.Implementation)
	if !ok {
		return fmt.Errorf("failed to create apiv1.Implementation from type %s", reflect.TypeOf(raw).String())
	}

	conf, err := ext.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get config from extension")
	}
	h.extensions[strings.ToLower(conf.Name)] = ext

	return nil
}

// getExtensionPath returns the path to an extension binary
func (h *Host) getExtensionPath(version, name string) string {
	homeDir, _ := os.UserHomeDir() //nolint:errcheck // Why: signature doesn't allow it, yet
	path := filepath.Join(homeDir, ".outreach", ".config", "stencil", "extensions", name, "@v", version, name)
	os.MkdirAll(filepath.Dir(path), 0o755) //nolint:errcheck // Why: signature doesn't allow it, yet
	return path
}

// buildFromLocal copies a local extension to it's stable path
func (h *Host) buildFromLocal(_ context.Context, filePath, name string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open extension")
	}

	dlPath := h.getExtensionPath("local", name)
	nf, err := os.Create(dlPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open extension path")
	}
	err = nf.Chmod(0755) //nolint:gocritic // Why: this is valid
	if err != nil {
		return "", errors.Wrap(err, "failed to chmod dest file")
	}

	_, err = io.Copy(nf, f)
	return dlPath, errors.Wrap(err, "failed to copy local extension to storage path")
}

// downloadFromRemote downloads a release from github and extracts it to disk
func (h *Host) downloadFromRemote(ctx context.Context, org, repo, name, version string) (string, error) {
	token, err := github.GetToken()
	if err != nil {
		return "", errors.Wrap(err, "failed to get github token")
	}

	gh := updater.NewGithubUpdater(ctx, token, org, repo)
	err = gh.Check(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to validate github client worked")
	}

	// TODO(jaredallard): Switch to using the gobox/pkg/github client
	// instead.
	var rel *gogithub.RepositoryRelease
	if version == "" {
		rel, err = gh.GetLatestVersion(ctx, "v0.0.0", false)
		if err != nil {
			return "", errors.Wrap(err, "failed to find latest extension version")
		}
		version = rel.GetTagName()
	} else {
		return "", fmt.Errorf("setting versions is not currently supported")
	}

	bin, cleanup, err := gh.DownloadRelease(ctx, rel, name, name)
	if cleanup != nil {
		cleanup()
	}
	if err != nil {
		return "", errors.Wrap(err, "failed to download extension")
	}

	dlPath := h.getExtensionPath(version, name)
	return dlPath, errors.Wrap(
		os.Rename(bin, dlPath),
		"failed to move downloaded extension",
	)
}
