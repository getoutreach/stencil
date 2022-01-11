package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/getoutreach/stencil/internal/tests"
	"github.com/getoutreach/stencil/pkg/codegen"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		if err := runTest(scanner.Bytes()); err != nil {
			logrus.WithError(err).Error("test failed")
			os.Exit(1)
		}
	}
	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Error("failed to read input")
		os.Exit(1)
	}
}

func runTest(input []byte) error { //nolint:funlen,gocyclo
	var t *tests.Test
	if err := json.Unmarshal(input, &t); err != nil {
		return errors.Wrap(err, "failed to parse test input")
	}

	m, err := t.Manifest()
	if err != nil {
		return errors.Wrap(err, "failed to decode test")
	}

	dir, cleanup, err := createRepoDir("my-repo", m)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}

	b := codegen.NewBuilder("my-repo", dir, logrus.New(), m, "")

	logrus.WithFields(logrus.Fields{
		"test.name": t.Name,
		"test.dir":  dir,
	}).Info("running stencil test")

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	defer func() {
		if err = os.Chdir(cwd); err != nil {
			logrus.WithError(err).Panic("failed to change directory")
		}
	}()

	if err := os.Chdir(dir); err != nil { //nolint:govet // Why: We're OK shadowing err
		return err
	}

	_, err = b.Run(context.Background())
	if err != nil {
		return err
	}

	if err := b.FormatFiles(context.Background()); err != nil {
		return err
	}

	return execTestInRepo()
}

func createRepoDir(repo string, m *configuration.ServiceManifest) (repodir string, cleanup func(), err error) { //nolint:funlen
	dir, err := ioutil.TempDir("", "codegen_test")
	if err != nil {
		return "", nil, err
	}
	cleanupFn := func() { os.RemoveAll(dir) }

	repodir = filepath.Join(dir, repo)
	if err := os.MkdirAll(repodir, os.ModePerm); err != nil { //nolint:govet // Why: We're OK shadowing err
		return "", cleanupFn, err
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = repodir
	if out, err := cmd.CombinedOutput(); err != nil { //nolint:govet // Why: We're OK shadowing err
		logrus.WithError(err).Errorf("failed to create git repository: %s", out)
		return "", cleanupFn, err
	}

	if os.Getenv("CI") != "" {
		//nolint:lll // Why: This is a long command line
		cmd = exec.Command("/usr/bin/env", "bash", "-c", "git config --global user.name stencil; git config --global user.email stencil@outreach.io")
		cmd.Dir = repodir
		if out, err := cmd.CombinedOutput(); err != nil { //nolint:govet // Why: We're OK shadowing err
			return "", cleanupFn, fmt.Errorf("failed to run git: %s", out)
		}
	}

	// Write the service.yaml
	b, err := yaml.Marshal(m)
	if err != nil {
		return "", cleanupFn, err
	}
	err = ioutil.WriteFile(filepath.Join(repodir, "service.yaml"), b, 0600)
	if err != nil {
		return "", cleanupFn, err
	}

	cmd = exec.Command("git", "commit", "-m", "chore: initial commit", "--allow-empty")
	cmd.Dir = repodir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", cleanupFn, fmt.Errorf("failed to create initial commit: %s", out)
	}

	cmd = exec.Command("git", "branch", "main")
	cmd.Dir = repodir
	if out, err := cmd.CombinedOutput(); err != nil {
		// If someone runs this locally they may have instructed git to create
		// a default branch, so we just log a warning here
		logrus.WithError(err).Warnf("failed to create main branch: %s", out)
	}

	return repodir, cleanupFn, nil
}

func execTestInRepo() error { //nolint:gocritic
	// add standard environments and disable GOFLAGS=-mod=readonly that
	// ./scripts/test.sh adds in CI because `go test ./...` below will
	// require the local go.mod to be writeable
	env := append(os.Environ(), "GOPRIVATE=github.com/getoutreach/*", "GO111MODULE=on", "GOFLAGS= ")

	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		logrus.WithError(err).Errorf("failed to run go generate: %s", out)
		return err
	}

	cmd = exec.Command("go", "test", "-tags=or_test", "./...")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		logrus.WithError(err).Errorf("failed to run go test: %s", out)
		return err
	}

	return nil
}
