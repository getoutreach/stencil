// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains configuration for stenciltest

// Package config contains configuration for stenciltest
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// configFileName is the name of the stenciltest configuration file
const configFileName = "stenciltest.yaml"

// config is the global configuration instance
//
//nolint:gochecknoglobals // Why: For tests only
var config = &Config{
	Validators: []Validator{},
}

// init is used to load the global configuration
//
//nolint:gochecknoinits // Why: For tests only
func init() {
	var err error
	config, err = loadConfig()
	if err != nil {
		panic("stenciltest: failed to load config: " + err.Error())
	}
}

// GetConfig returns the configuration for stenciltest
func GetConfig() *Config {
	return config
}

// Config contains configuration for stenciltest
type Config struct {
	// Validators is a list of global validators to run on all template
	// snapshots. This is useful for things like validating that all
	// templates are formatted correctly.
	Validators []Validator `yaml:"validators"`
}

// GetRepositoryDirectory returns the repository directory regardless
// of the current directory. This currently uses `GOMOD` to determine
// the root of the repository.
func GetRepositoryDirectory() (string, error) {
	b, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to determine path to manifest")
	}
	return strings.TrimSuffix(strings.TrimSpace(string(b)), "/go.mod"), nil
}

// loadConfig returns the stenciltest configuration for the current
// repository
func loadConfig() (*Config, error) {
	repoDir, err := GetRepositoryDirectory()
	if err != nil {
		return nil, err
	}
	var cfg Config

	if f, err := os.Open(filepath.Join(repoDir, configFileName)); err == nil {
		defer f.Close() //nolint:errcheck // Why: We don't care if this fails
		if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
			return nil, errors.Wrap(err, "failed to decode config")
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.Wrap(err, "failed to open config")
	}

	// augment the config with defaults if not set
	if cfg.Validators == nil {
		cfg.Validators = []Validator{}
	}

	// validate the config
	for i, v := range cfg.Validators {
		// Check if command AND func is not set, need one or the other
		if v.Command == "" && v.Func == nil {
			return nil, fmt.Errorf("validator %d: command empty", i)
		}

		// Check if command is set AND func is set, can't use both
		if v.Command != "" && v.Func != nil {
			return nil, fmt.Errorf("validator %d: command and func set, one or the other must be set but not both", i)
		}
	}

	return &cfg, nil
}
