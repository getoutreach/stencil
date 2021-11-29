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

	var conf map[string]interface{}
	err = yaml.NewDecoder(f).Decode(&conf)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse gh auth configuration")
	}

	if _, ok := conf["github.conf"]; !ok {
		return "", fmt.Errorf("failed to find host 'github.com' in gh auth config, try running 'gh auth login'")
	}

	conf, ok := conf["github.com"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("expected map[string]interface{} for github.com host, got %v", reflect.TypeOf(conf["github.com"]).String())
	}
	tokenInf, ok := conf["oauth_token"]
	if !ok {
		return "", fmt.Errorf("failed to find oauth_token in gh auth config, try running 'gh auth login'")
	}

	token, ok := tokenInf.(string)
	if !ok {
		return "", fmt.Errorf("expected string for oauth_token, got %s", reflect.TypeOf(token).String())
	}

	return cfg.SecretData(token), nil
}
