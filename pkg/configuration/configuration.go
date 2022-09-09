// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description.

// Package configuration implements configuration loading logic
// for stencil repositories and template repositories
package configuration

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ValidateNameRegexp is the regex used to validate the service's name
const ValidateNameRegexp = `^[_a-z][_a-z0-9-]*$`

// NewServiceManifest reads a service manifest from disk at the
// specified path, parses it, and returns the output.
func NewServiceManifest(path string) (*ServiceManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var s *ServiceManifest
	if err := yaml.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}

	if !ValidateName(s.Name) {
		return nil, fmt.Errorf("name field in %q was invalid", path)
	}

	return s, nil
}

// NewDefaultServiceManifest returns a parsed service manifest
// from a set default path on disk.
func NewDefaultServiceManifest() (*ServiceManifest, error) {
	return NewServiceManifest("service.yaml")
}

// ServiceManifest is a manifest used to describe a service and impact
// what files are included
type ServiceManifest struct {
	// Name is the name of the service
	Name string `yaml:"name"`

	// Modules are the template modules that this service depends
	// on and utilizes
	Modules []*TemplateRepository `yaml:"modules,omitempty"`

	// Versions is a map of versions of certain tools, this is used by templates
	// and will likely be replaced with something better in the future.
	Versions map[string]string `yaml:"versions,omitempty"`

	// Arguments is a map of arbitrary arguments to pass to the generator
	Arguments map[string]interface{} `yaml:"arguments"`

	// Replacements is a list of module names to replace their URI.
	// Expected format:
	// - local file: file://path/to/module
	// - remote file: https://github.com/getoutreach/stencil-base
	// - remote file w/ different protocol: git@github.com:getoutreach/stencil-base
	Replacements map[string]string `yaml:"replacements,omitempty"`
}

// TemplateRepository is a repository of template files.
type TemplateRepository struct {
	// Name is the name of this module. This should be a valid go import path
	Name string `yaml:"name"`

	// Deprecated: Use 'channel' instead, prerelease sets 'channel' to 'rc'.
	// Prerelease is a boolean indicating whether or not to consider prerelease versions
	Prerelease bool `yaml:"prerelease"`

	// Deprecated: Use name instead
	// URL is a full URL for a given module
	URL string `yaml:"url,omitempty"`

	// Channel is the channel to use for updates to this module
	// Defaults to "stable"
	Channel string `yaml:"channel,omitempty"`

	// Version is a semantic version or branch of the template repository
	// that should be downloaded if not set then the latest version is used.
	//
	// Version can also be a constraint as supported by the underlying resolver:
	// !!! <TODO link to gobox/pkg/cli/updater/resolver>
	Version string `yaml:"version,omitempty"`
}

// TemplateRepositoryManifest is a manifest of a template repository
type TemplateRepositoryManifest struct {
	// Name is the name of this template repository.
	// This must match the import path.
	Name string `yaml:"name"`

	// Modules are template repositories that this manifest requires
	Modules []*TemplateRepository `yaml:"modules"`

	// Type stores a comma-separated list of template repository types served by the current module.
	// Use the TemplateRepositoryTypes.Contains method to check.
	Type TemplateRepositoryTypes `yaml:"type,omitempty"`

	// PostRunCommand is a command to be ran after rendering and post-processors
	// have been ran on the project
	PostRunCommand []*PostRunCommandSpec `yaml:"postRunCommand,omitempty"`

	// Arguments are a declaration of arguments to the template generator
	Arguments map[string]Argument `yaml:"arguments,omitempty"`
}

// PostRunCommandSpec is the spec of a command to be ran and its
// friendly name
type PostRunCommandSpec struct {
	// Name is the name of the command being ran, used for UX
	Name string `yaml:"name"`

	// Command is the command to be ran, note: this is ran inside
	// of a bash shell.
	Command string `yaml:"command"`
}

// Argument is a user-input argument that can be passed to
// templates
type Argument struct {
	// Description is a description of this argument.
	Description string `yaml:"description"`

	// Required denotes this argument as required.
	Required bool `yaml:"required"`

	// Default is the default value for this argument if it's not set.
	// This cannot be set when required is true.
	Default interface{} `yaml:"default"`

	// Schema is a JSON schema, in YAML, for the argument.
	Schema map[string]interface{} `yaml:"schema"`

	// Deprecated fields below.

	// Deprecated: Use schema instead
	// Type declares the type of the argument. This is not implemented
	// yet, so is likely to change in the future.
	Type string `yaml:"type"`

	// Deprecated: Use schema instead.
	// Values is a list of possible values for this, if empty all input is
	// considered valid.
	Values []string `yaml:"values"`

	// From is a reference to an argument in another module, if this is
	// set, all other fields are ignored and instead the module referenced
	// field's are used instead. The name of the argument, the key in the map,
	// must be the same across both modules.
	From string `yaml:"from"`
}

// ValidateName ensures that the name of a service in the manifest
// fits the criteria we require.
func ValidateName(name string) bool {
	// This is more restrictive than the actual spec.  We're artificially
	// restricting ourselves to non-Unicode names because (in practice) we
	// probably don't support international characters very well, either.
	//
	// See also:
	// 	https://golang.org/ref/spec#Identifiers
	acceptableName := regexp.MustCompile(ValidateNameRegexp)
	return acceptableName.MatchString(name)
}
