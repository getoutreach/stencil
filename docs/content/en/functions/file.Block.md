---
title: file.Block
linktitle: file.Block
description: >
  Block returns the contents of a given block ###Block(name) Hello, world! ###EndBlock(name)
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
---

Block returns the contents of a given block \#\#\#Block\(name\) Hello\, world\! \#\#\#EndBlock\(name\)

```go-text-template
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

