---
title: stencil.Exists
linktitle: stencil.Exists
description: >
  Exists returns true if the file exists in the current directory
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

```go-text-template
{{- if stencil.Exists "myfile.txt" }}
{{ stencil.ReadFile "myfile.txt" }}
{{- end }}
```
