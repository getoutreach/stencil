---
title: stencil.SetGlobal
linktitle: stencil.SetGlobal
description: >
  
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

SetGlobal sets a global to be used in the context of the current template module repository\. This is useful because sometimes you want to define variables inside of a helpers template file after doing manifest argument processing and then use them within one or more template files to be rendered; however\, go templates limit the scope of symbols to the current template they are defined in\, so this is not possible without external tooling like this function\.


This template function stores \(and its inverse\, GetGlobal\, retrieves\) data that is not strongly typed\, so use this at your own risk and be averse to panics that could occur if you're using the data it returns in the wrong way\.


```go-text-template
{{- /* This writes a global into the current context of the template module repository */}}
{{ stencil.SetGlobal "IsGeorgeCool" true }}
```


