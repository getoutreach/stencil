---
title: file.SetContents
linktitle: file.SetContents
description: >
  SetContents sets the contents of file being rendered to the value
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---


This is useful for programmatic file generation within a template\.


```go-text-template
{{ file.SetContents "Hello, world!" }}
```


