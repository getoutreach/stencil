// Package vfs implements a layered filesystem for use with billy.Filesystem
package vfs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
)

// Compile time assertion
var _ billy.Filesystem = &LayeredFS{}

// LayeredFS implements the billy.Filesystem interface
type LayeredFS struct {
	filesystems []billy.Filesystem
}

// NewLayeredFile creates a layered file-system from a array
// of billy.Filesystem instances
func NewLayeredFS(filesystems ...billy.Filesystem) *LayeredFS {
	return &LayeredFS{
		filesystems: filesystems,
	}
}

// findFile finds a file across all filesystems, and returns a os.ErrNotExist
// if it's not found, and returns the filesystem it belongs to if it's found.
// If a file is found in multiple filesystems the lower indexed filesystem,
// e.g. the first filesystem provided provided to NewLayeredFS, is used
// first.
func (l *LayeredFS) findFile(path string) (billy.Filesystem, error) {
	for _, fs := range l.filesystems {
		if _, err := fs.Stat(path); err == nil {
			return fs, nil
		}
	}

	return nil, os.ErrNotExist
}

func (l *LayeredFS) Create(path string) (billy.File, error) {
	return nil, fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Open(path string) (billy.File, error) {
	fs, err := l.findFile(path)
	if err != nil {
		return nil, err
	}

	return fs.Open(path)
}

func (l *LayeredFS) OpenFile(path string, flag int, perm os.FileMode) (billy.File, error) {
	return nil, fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Stat(path string) (os.FileInfo, error) {
	fs, err := l.findFile(path)
	if err != nil {
		return nil, err
	}

	return fs.Stat(path)
}

func (l *LayeredFS) Rename(oldPath, newPath string) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Remove(path string) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (l *LayeredFS) TempFile(dir, prefix string) (billy.File, error) {
	return nil, fmt.Errorf("unsupported on layered filesystem")
}

// ReadDir will read a directory across all available filesystems, and deduplicate
func (l *LayeredFS) ReadDir(dir string) ([]os.FileInfo, error) {
	files := make(map[string]os.FileInfo)

	var foundDir bool
	for _, fs := range l.filesystems {
		filelist, err := fs.ReadDir(dir)
		if err != nil {
			// TODO(jaredallard): better error handling here
			continue
		}

		// if we had no error at one point, we set foundDir
		// to make sure we return the correct error type across
		// all of the filesystems
		foundDir = true
		for _, f := range filelist {
			files[f.Name()] = f
		}
	}

	// if we didn't find it at, assume ErrNotExist was the reason
	if !foundDir {
		return nil, os.ErrNotExist
	}

	// convert our de-duplication hash map back into a list
	list := make([]os.FileInfo, 0)
	for _, f := range files {
		list = append(list, f)
	}

	return list, nil
}

func (l *LayeredFS) MkdirAll(dir string, perm os.FileMode) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

///
// Symlink
///

func (l *LayeredFS) Lstat(path string) (os.FileInfo, error) {
	fs, err := l.findFile(path)
	if err != nil {
		return nil, err
	}

	return fs.Lstat(path)
}

func (l *LayeredFS) Symlink(target, link string) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Readlink(path string) (string, error) {
	fs, err := l.findFile(path)
	if err != nil {
		return "", err
	}

	return fs.Readlink(path)
}

///
// Chmod
///

func (l *LayeredFS) Chmod(name string, mode os.FileMode) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Lchown(name string, uid, gid int) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Chown(name string, uid, gid int) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Chtimes(name string, atime, mtime time.Time) error {
	return fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Chroot(path string) (billy.Filesystem, error) {
	return nil, fmt.Errorf("unsupported on layered filesystem")
}

func (l *LayeredFS) Root() string {
	return ""
}
