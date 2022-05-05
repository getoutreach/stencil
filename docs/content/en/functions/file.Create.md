---
title: file.Create
linktitle: file.Create
description: >
  Create creates a new file that is rendered by the current template. If the template has a single file with no contents this file replaces it.
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
---

Create creates a new file that is rendered by the current template\. If the template has a single file with no contents this file replaces it\.

```go-text-template
{{- define "command" }}
package main

import "fmt"

func main() {
  fmt.Println("hello, world!")
}

{{- end }}

# Generate a "<commandName>.go" file for each command in .arguments.commands
{{- range $_, $commandName := (stencil.Arg "commands") }}
{{- file.Create (printf "cmd/%s.go" $commandName) 0600 now }}
{{- stencil.ApplyTemplate "command" | file.SetContents }}
{{- end }}
```

