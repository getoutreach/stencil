// Package configuration implements configuration loading logic
// for stencil repositories and template repositories
package configuration

import (
	"os"

	"gopkg.in/yaml.v2"
)

// NewServiceManifest reads a service manifest from disk at the
// specified path, parses it, and returns the output.
func NewServiceManifest(path string) (*ServiceManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var s *ServiceManifest
	err = yaml.NewDecoder(f).Decode(&s)
	return s, err
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
	Modules []*TemplateRepository `yaml:"modules"`

	// Versions is a map of versions of certain tools, this is used by templates
	// and will likely be replaced with something better in the future.
	Versions map[string]string `yaml:"versions,omitempty"`

	// Arguments is a map of arbitrary arguments to pass to the generator
	Arguments map[string]interface{} `yaml:"arguments"`
}

// TemplateRepository is a repository of template files.
type TemplateRepository struct {
	// URL is the fully qualified URL that is able to access the templates
	// and manifest.
	URL string `yaml:"url"`

	// Version is a semantic version of the template repository that should be downloaded
	// if not set then the latest version is used.
	Version string `yaml:"version"`
}

// TemplateRepositoryManifest is a manifest of a template repository
type TemplateRepositoryManifest struct {
	// Name is the name of this template repository.
	// This is likely to be used in the future.
	Name string `yaml:"name"`

	// Modules are template repositories that this manifest requires
	Modules []*TemplateRepository `yaml:"modules"`

	// PostRunCommand is a command to be ran after rendering and post-processors
	// have been ran on the project
	PostRunCommand []*PostRunCommandSpec `yaml:"postRunCommand"`

	// Arguments are a declaration of arguments to the template generator
	Arguments map[string]Argument
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

type Argument struct {
	// Required denotes this argument as required.
	Required bool `yaml:"required"`

	// Type declares the type of the argument. This is not implemented
	// yet, so is likely to change in the future.
	Type string `yaml:"type"`

	// Values is a list of possible values for this, if empty all input is
	// considered valid.
	Values []string `yaml:"values"`

	// Description is a description of this argument. Optional.
	Description string `yaml:"description"`
}
