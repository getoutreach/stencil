---
{{- $description := "" }}
{{- /* Use the first paragraph in a comment as the description */}}
{{- range .Doc.Blocks }}
  {{- if eq .Kind "header" }}
    {{- $description = .Text }}
    {{- break}}
  {{- end }}
{{- end }}
{{- $namespace := "" }}
{{- if eq .Receiver "*TplStencil" }}
{{- $namespace = "stencil" }}
{{- else if eq .Receiver "*TplFile" }}
{{- $namespace = "file" }}
{{- end }}
title: {{ $namespace }}.{{ .Name }}
linktitle: {{ $namespace }}.{{ .Name }}
description: >
  {{ $description }}
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

{{ range .Doc.Blocks }}
  {{- /* Use text but not the description */}}
	{{- if eq .Kind "paragraph" }}
		{{- paragraph .Text }}
	{{- else if eq .Kind "code" }}
		{{- codeBlock "go-text-template" .Text }}
	{{- end }}
{{ end }}