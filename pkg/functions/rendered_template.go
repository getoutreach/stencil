// Copyright 2021 Outreach Corporation. All Rights Reserved.

// Description: Holds all context for a rendered template file
package functions

import (
	"io"
)

// RenderedTemplate is a file that has been processed by stencil
type RenderedTemplate struct {
	io.Reader

	// Skipped marks if this file should be skipped or not
	Skipped bool

	// SkipReason is the reason this file was skipped
	// only valid when Skip is true and is optional
	SkipReason string

	// Deleted marks if this file has been deleted or not
	Deleted bool

	// Path is the path this template should be written to
	// Note: This is a relative path, b.Dir is injected
	// at the writeFile step as the base.
	Path string

	// Warnings is an array of warnings that were created
	// while rendering this template
	Warnings []string
}

// AddDeprecationNotice adds a warning to the rendered template file that
// will be returned to the renderer. When ran via stencil this will be outputted
// to the console.
//
//  {{- if thingWeShouldNotSupport }}
//	{{- file.AddDeprecationNotice "No longer supported"}}
//  {{- end }}
func (rt *RenderedTemplate) AddDeprecationNotice(msg string) string {
	if rt.Warnings == nil {
		rt.Warnings = []string{msg}
		return ""
	}

	rt.Warnings = append(rt.Warnings, msg)
	return ""
}

// MarkStatic marks this rendered template as being static. This file
// will only be written once, and future changes to this template will not be reflected.
//
//   {{- file.MarkStatic }}
func (rt *RenderedTemplate) MarkStatic() string {
	rt.Skipped = true
	rt.SkipReason = "File was marked as static"
	return ""
}

// Skip marks this template as skipped with a user-providable
// reason.
//
//   {{- if someReason }}
//   {{- file.Skip "LEEEEEEERROOOYYYY JENKIIIINNNSSS" }}
//   {{- end }}
func (rt *RenderedTemplate) Skip(reason string) string {
	rt.Skipped = true
	rt.SkipReason = reason
	return ""
}

// Delete marks this rendered template as being deleted and will be
// removed by the renderer if it exists
//
//   {{- file.Delete }}
func (rt *RenderedTemplate) Delete() string {
	rt.Deleted = true
	return ""
}

// SetPath sets the path of this template which changes where the renderer
// will write this file in the context of the running directory
//
//   {{- $appName := (stencil.Arg "name") }}
//   {{- file.SetPath (printf "cmd/%s/%s.go" $appName $appName)}}
func (rt *RenderedTemplate) SetPath(name string) string {
	rt.Path = name
	return ""
}
