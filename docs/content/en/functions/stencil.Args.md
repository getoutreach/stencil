---
title: stencil.Args
linktitle: stencil.Args
description: >
  
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

Deprecated: Use Arg instead\. Args returns all arguments passed to stencil from the service's manifest


Note: This doesn't set default values and is instead representative of \_all\_ data passed in its raw form\.


This is deprecated and will be removed in a future release\.


```go-text-template
{{- (stencil.Args).name }}
```


