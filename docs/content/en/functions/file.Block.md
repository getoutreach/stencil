---
title: file.Block
linktitle: file.Block
description: >
  Block returns the contents of a given block
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---


```go-text-template
###Block(name)
Hello, world!
###EndBlock(name)

###Block(name)
{{- /* Only output if the block is set */}}
{{- if not (empty (file.Block "name")) }}
{{ file.Block "name" }}
{{- end }}
###EndBlock(name)

###Block(name)
{{ - /* Short hand syntax, but adds newline if no contents */}}
{{ file.Block "name" }}
###EndBlock(name)
```


