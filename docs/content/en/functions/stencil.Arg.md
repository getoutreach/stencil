---
title: stencil.Arg
linktitle: stencil.Arg
description: >
  Arg returns the value of an argument in the service's manifest
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

```go-text-template
{{- stencil.Arg "name" }}
```

Note: Using \`stencil\.Arg\` with no path returns all arguments and is equivalent to \`stencil\.Args\`\. However\, that is DEPRECATED along with \`stencil\.Args\` as it doesn't provide default types\, or check the JSON schema\, or track which module calls what argument\.
