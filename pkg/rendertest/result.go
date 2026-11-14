// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the Result of a render.

package rendertest

// Result is the outcome of rendering a template. Its files are the files
// of the template under test, not those of helper templates.
type Result struct {
	// err is the render error
	err error

	// files are the rendered files that were not deleted or skipped
	files []*FileResult

	// deletedPaths are the paths of the files that were deleted
	deletedPaths []string

	// skippedPaths are the paths of the files that were skipped
	skippedPaths []string

	// postRunCommands are the post-run commands that would run
	postRunCommands []string

	// hookValues are the module hook values after the render
	hookValues map[string][]any
}

// Error returns the render error, or nil if the render succeeded.
func (r *Result) Error() error {
	return r.err
}

// Files returns the rendered files, other than deleted and skipped files.
func (r *Result) Files() []*FileResult {
	return r.files
}

// File returns the rendered file at path, or nil if there is none.
func (r *Result) File(path string) *FileResult {
	for _, f := range r.files {
		if f.path == path {
			return f
		}
	}
	return nil
}

// DeletedPaths returns the paths of the files that the template deleted.
func (r *Result) DeletedPaths() []string {
	return r.deletedPaths
}

// SkippedPaths returns the paths of the files that the template skipped.
func (r *Result) SkippedPaths() []string {
	return r.skippedPaths
}

// PostRunCommands returns the post-run commands of all modules, in order.
// They are recorded, not run.
func (r *Result) PostRunCommands() []string {
	return r.postRunCommands
}

// HookValues returns the values of all module hooks after the render, by
// module and hook name joined with "/", for example
// "github.com/getoutreach/stencil-base/myHook". It includes the values set
// with TemplateBuilder.WithHooks.
func (r *Result) HookValues() map[string][]any {
	return r.hookValues
}
