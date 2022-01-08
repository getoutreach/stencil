// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description

// Package github implements helpers accessing github
// via the gh cli
package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// githubKey is the standard host used for github.com
// in the gh config
const githubKey = "github.com"

// ghInstallInstructions denotes how to install the gh cli
var ghInstallInstructions = map[string]string{
	"windows": "Please run via WSL2",
	"linux":   "Download at https://github.com/cli/cli/releases",
	"darwin":  "Install via brew: brew install gh",
}

// GetGHToken gets a token from gh, or informs the user how to setup
// a github token via gh, or install gh if not found
func GetGHToken() (cfg.SecretData, error) {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return "", errors.Wrapf(err, "failed to find gh: %s", ghInstallInstructions[runtime.GOOS])
	} else if ghPath == "" {
		return "", fmt.Errorf("failed to find gh: %s", ghInstallInstructions[runtime.GOOS])
	}

	cmd := exec.Command("gh", "auth", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "failed to ensure we had a valid Github token: %s", out)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to get user home directory")
	}

	ghAuthPath := filepath.Join(homeDir, ".config", "gh", "hosts.yml")
	f, err := os.Open(ghAuthPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read gh auth configuration, try running 'gh auth login'")
	}
	defer f.Close()

	var conf map[string]interface{}
	err = yaml.NewDecoder(f).Decode(&conf)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse gh auth configuration")
	}

	if _, ok := conf[githubKey]; !ok {
		return "", fmt.Errorf("failed to find host '%s' in gh auth config, try running 'gh auth login'", githubKey)
	}

	realConf, ok := conf[githubKey].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("expected map[interface{}]interface{} for %s host, got %v", githubKey, reflect.ValueOf(conf[githubKey]).String())
	}

	tokenInf, ok := realConf["oauth_token"]
	if !ok {
		return "", fmt.Errorf("failed to find oauth_token in gh auth config, try running 'gh auth login'")
	}

	token, ok := tokenInf.(string)
	if !ok {
		return "", fmt.Errorf("expected string for oauth_token, got %s", reflect.ValueOf(token).String())
	}

	return cfg.SecretData(token), nil
}
