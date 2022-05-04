---
title: dict
description: Creates a dictionary from a list of key and value pairs.
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
keywords: [dictionary]
signature: ["dict KEY VALUE [KEY VALUE]..."]
workson: []
relatedfuncs: []
deprecated: false
aliases: []
---

`dict` is especially useful for passing more than one value to a partial template.

Note that the `key` should be a string.

```go-text-template
{{ $m := dict "k "value" }}
```

### Multiple Keys

```go-text-template
{{ $m := dict "k1" "v1" "k2" "v2" }}
```
