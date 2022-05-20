---
title: file.SetPath
linktitle: file.SetPath
description: >
  SetPath changes the path of the current file being rendered
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

```go-text-template
{{ $_ := file.SetPath "new/path/to/file.txt" }}
```

Note: The $\_ is required to ensure \<nil\> isn't outputted into the template\.
