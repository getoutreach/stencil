---
{{- $description := "" }}
{{- range .Doc.Blocks }}
{{- if eq .Kind "paragraph" }}
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
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
---

{{ with .Doc }}
{{- range .Blocks -}}
	{{- if eq .Kind "paragraph" -}}
		{{- paragraph .Text -}}
	{{- else if eq .Kind "code" -}}
		{{- codeBlock "go-text-template" .Text -}}
	{{- else if eq .Kind "header" -}}
		{{- header .Level .Text -}}
	{{- end -}}
{{- end -}}
{{- end }}