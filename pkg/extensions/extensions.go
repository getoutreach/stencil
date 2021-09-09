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
	"text/template"

	"github.com/getoutreach/gobox/pkg/updater"
	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/getoutreach/stencil/pkg/extensions/github"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	giturls "github.com/whilp/git-urls"

	gogithub "github.com/google/go-github/v34/github"
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

// GetTemplateFunctions returns a function map from the available
// plugins.
func (h *Host) GetTemplateFunctions(ctx context.Context) (template.FuncMap, error) {
	funcMap := map[string]interface{}{}
	for name, ext := range h.extensions {
		funcs, err := ext.GetTemplateFunctions()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get template functions from plugin '%s'", name)
		}

		for _, f := range funcs {
			funcKey := fmt.Sprintf("extensions.%s.%s", name, f.Name)

			// create a new function based on the arguments provided
			// that calls the rpc
			tfunc := reflect.MakeFunc(reflect.FuncOf(f.Arguments, []reflect.Type{nil}, false), func(reflectArgs []reflect.Value) []reflect.Value {
				args := make([]interface{}, len(reflectArgs))
				for i, v := range reflectArgs {
					args[i] = v.Interface()
				}

				iresp, err := ext.ExecuteTemplateFunction(&apiv1.TemplateFunctionExec{
					Name:      f.Name,
					Arguments: args,
					// TODO: Figure out how to inject stencil/file information here
				})
				if err != nil {
					return []reflect.Value{reflect.ValueOf(nil), reflect.ValueOf(err)}
				}

				// convert the response from an interface into reflect.Value
				// to satisfy MakeFunc
				resp := make([]reflect.Value, len(iresp))
				for i, inf := range iresp {
					resp[i] = reflect.ValueOf(inf)
				}

				return resp
			})
			funcMap[funcKey] = tfunc.Interface()
		}
	}

	return funcMap, nil
}

// RegisterExtension registers a ext from a given source
// and compiles/downloads it. A client is then created
// that is able to communicate with the ext.
func (h *Host) RegisterExtension(ctx context.Context, source, name, version string) error {
	u, err := giturls.Parse(source)
	if err != nil {
		return errors.Wrap(err, "failed to parse extension URL")
	}

	var extPath string
	if u.Scheme == "file" {
		extPath, err = h.buildFromLocal(ctx, u.Path, name)
	} else {
		extPath, err = h.downloadFromRemote(ctx, source, name, version)
	}
	if err != nil {
		return errors.Wrap(err, "failed to setup extension")
	}

	// create a connection to the extension
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  apiv1.Version,
			MagicCookieKey:   apiv1.CookieKey,
			MagicCookieValue: apiv1.CookieValue,
		},
		Plugins: map[string]plugin.Plugin{
			"extension": &apiv1.ExtensionPlugin{},
		},
		Cmd: exec.CommandContext(ctx, extPath),
	})

	rpcClient, err := client.Client()
	if err != nil {
		return errors.Wrap(err, "failed to create connection to extension")
	}

	raw, err := rpcClient.Dispense("extension")
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
func (h *Host) getExtensionPath(version string, name string) string {
	homeDir, _ := os.UserHomeDir()
	path := filepath.Join(homeDir, ".outreach", ".config", "stencil", "extensions", name, "@v", version, name)
	os.MkdirAll(path, 0755)
	return path
}

// buildFromLocal copies a local extension to it's stable path
func (h *Host) buildFromLocal(ctx context.Context, filePath, name string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open extension")
	}

	dlPath := h.getExtensionPath("local", name)
	nf, err := os.Create(dlPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open extension path")
	}

	_, err = io.Copy(nf, f)
	return dlPath, errors.Wrap(err, "failed to copy local extension to storage path")
}

// downloadFromRemote downloads a release from github and extracts it to disk
func (h *Host) downloadFromRemote(ctx context.Context, source, name, version string) (string, error) {
	token, err := github.GetGHToken()
	if err != nil {
		return "", errors.Wrap(err, "failed to get github token")
	}

	gh := updater.NewGithubUpdater(ctx, token, "", "")
	err = gh.Check(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to validate github client worked")
	}

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
