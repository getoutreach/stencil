---
title: stencil.GetGlobal
linktitle: stencil.GetGlobal
description: >
  
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---

GetGlobal retrieves a global variable set by SetGlobal\. The data returned from this function is unstructured so by averse to panics \- look at where it was set to ensure you're dealing with the proper type of data that you think it is\.


```go-text-template
{{- /* This retrieves a global from the current context of the template module repository */}}
{{ $isGeorgeCool := stencil.GetGlobal "IsGeorgeCool" }}
```


