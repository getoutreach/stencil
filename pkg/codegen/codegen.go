// Package codegen has code generators for Go projects
//
// This is intended for use with stencil
//
// Builder:Build is the main entry point and this uses the fields of
// the Manifest and the builder itself to first generate a list of
// files.  This is done by running the template files.json.tpl.
//
// The leaf nodes of the output JSON are expected to be template =>
// local file mappings.  These are used to create the corresponding
// local files.
//
// All the templates have access to the manifest and different formats
// of the app name (suitable for different purposes)
package codegen

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/stencil/internal/vfs"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/processors"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	ErrNotAFile           = errors.New("not a file")
	ErrNoHeadBranch       = errors.New("failed to find a head branch, does one exist?")
	ErrNoRemoteHeadBranch = errors.New("failed to get head branch from remote origin")

	blockPattern = regexp.MustCompile(`\w*(///|###|<!---)\s*([a-zA-Z ]+)\(([a-zA-Z ]+)\)`)
	headPattern  = regexp.MustCompile(`HEAD branch: ([[:alpha:]]+)`)

	// versionPattern ensures versions have at least a major and a minor.
	//
	// For examples, see https://regex101.com/r/ajHtpK/1
	versionPattern = regexp.MustCompile(`^\d+\.\d+[.\d+]*$`)
)

type Builder struct {
	Branch    string
	Repo      string
	Dir       string
	Manifest  *configuration.ServiceManifest
	GitRepoFs billy.Filesystem

	Processors *processors.Table

	log logrus.FieldLogger

	sshKeyPath  string
	accessToken cfg.SecretData

	// set by Run
	postRunCommands []*configuration.PostRunCommandSpec
}

func NewBuilder(repo, dir string, s *configuration.ServiceManifest, sshKeyPath string, accessToken cfg.SecretData) *Builder {
	return &Builder{
		Repo:       repo,
		Dir:        dir,
		Manifest:   s,
		Processors: processors.New(),

		sshKeyPath:  sshKeyPath,
		accessToken: accessToken,

		postRunCommands: make([]*configuration.PostRunCommandSpec, 0),
	}
}

// Run generates the list of files using the files.json.tpl template
// and then generates the individual files as well. Returned is a list of
// warnings and if an error occurred that rendered it impossible to run stencil
func (b *Builder) Run(ctx context.Context, log logrus.FieldLogger) ([]string, error) {
	b.log = log

	if err := b.processManifest(); err != nil {
		return nil, errors.Wrap(err, "failed to process service manifest")
	}

	b.log.Info("Fetching dependencies")
	fetcher := NewFetcher(b.log, b.Manifest, b.sshKeyPath, b.accessToken)
	fs, manifests, err := fetcher.CreateVFS()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create vfs")
	}
	b.GitRepoFs = fs

	for _, m := range manifests {
		b.postRunCommands = append(b.postRunCommands, m.PostRunCommand...)
	}

	return b.GenerateFiles(ctx, fs)
}

// processManifest handles processing any fields in the manifest, i.e validation
func (b *Builder) processManifest() error {
	for resource, version := range b.Manifest.Versions {
		if !versionPattern.MatchString(version) {
			return fmt.Errorf("resource \"%s\" must have at least a major and minor version (format: MAJOR.MINOR.PATCH)", resource)
		}
	}

	return nil
}

func (b *Builder) FormatFiles(ctx context.Context) error {
	b.log.Info("Running post-run commands")
	for _, prc := range b.postRunCommands {
		b.log.Infof(" - %s", prc.Name)
		cmd := exec.CommandContext(ctx, "env", "bash", "-c", prc.Command) //nolint:gosec // Why: We have to
		cmd.Dir = b.Dir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return errors.Wrapf(err, "failed to run '%s'", prc.Command)
		}
	}

	return nil
}

// GenerateFiles generates local files from the templates inside of the specific git
// repository from the service.yaml.
func (b *Builder) GenerateFiles(ctx context.Context, fs billy.Filesystem) ([]string, error) {
	data, err := b.makeTemplateParameters(ctx)
	if err != nil {
		return nil, err
	}

	warnings := make([]string, 0)
	return warnings, vfs.Walk(fs, "", func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", path)
		}

		// Skip files without a .tpl extension
		if filepath.Ext(path) != ".tpl" {
			return nil
		}

		contents, err := b.FetchTemplate(ctx, path)
		if err != nil {
			return errors.Wrap(err, "failed to fetch template")
		}

		byt, err := ioutil.ReadAll(contents)
		if err != nil {
			return errors.Wrap(err, "failed to read file into memory")
		}

		w, err := b.WriteTemplate(ctx, path, string(byt), data)
		if err != nil {
			return errors.Wrap(err, "failed to write template")
		}

		warnings = append(warnings, w...)

		return nil
	})
}

// determineHeadBranch determines the remote head branch
func (b *Builder) determineHeadBranch(ctx context.Context) (string, error) {
	r, err := git.PlainOpen(b.Dir)
	if err != nil {
		return "", errors.Wrap(err, "failed to open directory as a repository")
	}

	_, err = r.Remote("origin")
	if err != nil {
		// loop through the local branchs
		candidates := []string{"main", "master"}
		for _, branch := range candidates {
			_, err := r.Reference(plumbing.NewBranchReferenceName(branch), true) //nolint:govet
			if err == nil {
				return branch, nil
			}
		}

		// we couldn't find one
		return "", ErrNoHeadBranch
	}

	// we found an origin reference, figure out the HEAD
	cmd := exec.CommandContext(ctx, "git", "remote", "show", "origin")
	cmd.Dir = b.Dir
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get head branch from remote origin")
	}

	matches := headPattern.FindStringSubmatch(string(out))
	if len(matches) != 2 {
		return "", ErrNoRemoteHeadBranch
	}

	return matches[1], nil
}

// makeTemplateParameters creates the map to be provided to the templates.
func (b *Builder) makeTemplateParameters(ctx context.Context) (map[string]interface{}, error) { //nolint:funlen
	headBranch, err := b.determineHeadBranch(ctx)
	if err == ErrNoHeadBranch {
		headBranch = "main"
	} else if err != nil {
		return nil, err
	}

	boxConf, err := box.LoadBox()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load box config")
	}

	return map[string]interface{}{
		"Metadata": map[string]string{
			"Generator": "stencil",
			"Version":   app.Version,
		},

		"Repository": map[string]string{
			"HeadBranch": headBranch,
		},

		"Box": boxConf,

		"Manifest":  b.Manifest,
		"Arguments": b.Manifest.Arguments,
	}, nil
}

// FetchTemplate fetches a template from a git repository
func (b *Builder) FetchTemplate(ctx context.Context, filePath string) (io.Reader, error) {
	f, err := b.GitRepoFs.Open(filePath)
	return f, errors.Wrap(err, filePath)
}

// HasDeviations looks for deviation blocks in a file, returning true if they exist
func (b *Builder) HasDeviations(_ context.Context, filePath string) bool {
	// Search for any commands that are inscribed in the file.
	// Currently we use Block and EndBlock to allow for
	// arbitrary data payloads to be saved across runs of stencil.
	// Eventually we might want to support 3 way merge instead
	f, err := os.Open(filePath)
	if err == nil {
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for i := 0; scanner.Scan(); i++ {
			line := scanner.Text()
			matches := blockPattern.FindStringSubmatch(line)

			// 1: Comment (###|///)
			// 2: Command
			// 3: Argument to the command
			if len(matches) >= 2 {
				cmd := matches[2]
				if strings.EqualFold(cmd, "deviation") {
					return true
				}
			}
		}
	}

	return false
}

// WriteTemplate writes the template to disk
func (b *Builder) WriteTemplate(ctx context.Context, filePath, contents string, args map[string]interface{}) ([]string, error) { //nolint:funlen,gocyclo,gocritic
	// Search for any commands that are inscribed in the file.
	// Currently we use Block and EndBlock to allow for
	// arbitrary data payloads to be saved across runs of stencil.
	// Eventually we might want to support 3 way merge instead
	f, err := os.Open(filePath)
	if err == nil {
		defer f.Close()

		var curBlockName string
		scanner := bufio.NewScanner(f)
		for i := 0; scanner.Scan(); i++ {
			line := scanner.Text()
			matches := blockPattern.FindStringSubmatch(line)
			isCommand := false

			// 1: Comment (###|///)
			// 2: Command
			// 3: Argument to the command
			if len(matches) == 4 {
				cmd := matches[2]
				isCommand = true

				switch cmd {
				case "Block":
					blockName := matches[3]

					if curBlockName != "" {
						return nil, fmt.Errorf("invalid Block when already inside of a block, at %s:%d", filePath, i)
					}
					curBlockName = blockName
				case "EndBlock":
					blockName := matches[3]

					if blockName != curBlockName {
						return nil, fmt.Errorf(
							"invalid EndBlock, found EndBlock with name '%s' while inside of block with name '%s', at %s:%d",
							blockName, curBlockName, filePath, i,
						)
					}

					if curBlockName == "" {
						return nil, fmt.Errorf("invalid EndBlock when not inside of a block, at %s:%d", filePath, i)
					}

					curBlockName = ""
				default:
					isCommand = false
				}
			}

			// we skip lines that had a recognized command in them, or that
			// aren't in a block
			if isCommand || curBlockName == "" {
				continue
			}

			// add the line we processed to the current block we're in
			// and account for having an existing curVal or not. If we
			// don't then we assign curVal to start with the line we
			// just found.
			curVal, ok := args[curBlockName]
			if ok {
				args[curBlockName] = curVal.(string) + "\n" + line
			} else {
				args[curBlockName] = line
			}
		}
	}

	warnings := make([]string, 0)

	if b.HasDeviations(ctx, filePath) {
		warnings = append(warnings, fmt.Sprintf("SKIPPED: '%s' had deviations and will not be re-generated", filePath))
		return warnings, nil
	}

	templates, err := b.renderTemplate(filePath, contents, args)
	if err != nil {
		return nil, err
	}

	for _, renderedTemplate := range templates {
		if len(renderedTemplate.Warnings) > 0 {
			warnings = append(warnings, renderedTemplate.Warnings...)
		}
		if renderedTemplate.Skipped {
			return warnings, nil
		}
		if renderedTemplate.Deleted {
			return warnings, os.RemoveAll(renderedTemplate.Path)
		}
		if renderedTemplate.Path != "" {
			filePath = renderedTemplate.Path
		}

		existingF, err := os.Open(filePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, errors.Wrap(err, "failed to open existing file")
		}
		defer existingF.Close()

		existingFile := processors.NewFile(existingF, filePath)
		templateFile := processors.NewFile(renderedTemplate, filePath)

		processedFile, err := b.Processors.Process(false, existingFile, templateFile)
		if err == nil {
			// Use the processor reader instead
			renderedTemplate.Reader = processedFile
		} else if err != nil && err != processors.ErrNotProcessable {
			return nil, errors.Wrap(err, "failed to process file")
		}

		absFilePath := path.Join(b.Dir, filePath)

		action := "Updated"
		if _, err := os.Stat(absFilePath); os.IsNotExist(err) { // nolint: govet,gocritic
			action = "Created"
		}

		perms := os.FileMode(0644)
		if strings.HasSuffix(filePath, ".sh") {
			perms = os.FileMode(0744)
		}
		filePath = strings.TrimSuffix(filePath, ".tpl")

		b.log.Infof("%s file '%s'", action, filePath)
		if err := b.writeFile(filePath, renderedTemplate, perms); err != nil {
			return nil, errors.Wrapf(err, "error creating file '%s'", absFilePath)
		}
	}

	return warnings, nil
}

//nolint:gocritic,funlen
func (b *Builder) renderTemplate(fileName, contents string,
	args map[string]interface{}) ([]*RenderedTemplate, error) {
	srcRendered := &RenderedTemplate{}

	tmpl := template.New(fileName)
	st := &Stencil{tmpl, b.Manifest, make([]*RenderedTemplate, 0), srcRendered}

	nargs := make(map[string]interface{})
	for k, v := range args {
		nargs[k] = v
	}

	funcs := templateFunctions
	funcs["stencil"] = func() *Stencil { return st }
	funcs["file"] = func() *RenderedTemplate { return st.File }

	tmpl, err := tmpl.Funcs(sprig.TxtFuncMap()).Funcs(funcs).Parse(contents)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nargs)
	srcRendered.Reader = &buf
	return append(st.Files, srcRendered), err
}

func (b *Builder) writeFile(fileName string, tf io.Reader, perm os.FileMode) error {
	fileName = filepath.Join(b.Dir, fileName)
	err := os.MkdirAll(filepath.Dir(fileName), os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	err = f.Chmod(perm)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, tf)
	return err
}
