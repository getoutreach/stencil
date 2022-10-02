---
title: stencil.ReadBlocks
linktitle: stencil.ReadBlocks
description: >

date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

ReadBlocks parses a file and attempts to read the blocks from it\, and their data\.

As a special case\, if the file does not exist\, an empty map is returned instead of an error\.

\*\*NOTE\*\*: This function does not guarantee that blocks are able to be read during runtime\. for example\, if you try to read the blocks of a file from another module there is no guarantee that that file will exist before you run this function\. Nor is there the ability to tell stencil to do that \(stencil does not have any order guarantees\)\. Keep that in mind when using this function\.

```go-text-template
{{- $blocks := stencil.ReadBlocks "myfile.txt" }}
{{- range $name, $data := $blocks }}
  {{- $name }}
  {{- $data }}
{{- end }}
```
