package stencil

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// LockfileName is the name of the lockfile used by stencil
	LockfileName    = "stencil.lock"
	oldLockfileName = "bootstrap.lock"
)

type Lockfile struct {
	// Version correlates to the version of bootstrap
	// that generated this file.
	Version string `yaml:"version"`

	// Generated was the last time this file was modified
	Generated time.Time `yaml:"generated"`
}

// LoadLockfile loads a lockfile from a bootstrap
// repository path
func LoadLockfile(path string) (*Lockfile, error) {
	f, err := os.Open(filepath.Join(path, LockfileName))
	if errors.Is(err, os.ErrNotExist) {
		f, err = os.Open(oldLockfileName)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	var lock *Lockfile
	err = yaml.NewDecoder(f).Decode(&lock)
	return lock, err
}
