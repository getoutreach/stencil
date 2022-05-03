---
title: stencil.ApplyTemplate
linktitle: stencil.ApplyTemplate
description: >
  ApplyTemplate executes a template inside of the current module that belongs to the actively rendered template. It does not support rendering a template from another module.
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
---

ApplyTemplate executes a template inside of the current module that belongs to the actively rendered template\. It does not support rendering a template from another module\.

```go-text-template
{{- define "command"}}
package main

import "fmt"

func main() {
  fmt.Println("hello, world!")
}

{{- end }}

{{- stencil.ApplyTemplate "command" | file.SetContents }}
```

