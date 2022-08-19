---
title: stencil.GetModuleHook
linktitle: stencil.GetModuleHook
description: >
  GetModuleHook returns a module block in the scope of this module
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---


This is incredibly useful for allowing other modules to write to files that your module owns\. Think of them as extension points for your module\. The value returned by this function is always a \[\]interface\{\}\, aka a list\.


```go-text-template
{{- /* This returns a []interface{} */}}
{{ $hook := stencil.GetModuleHook "myModuleHook" }}
{{- range $hook }}
  {{ . }}
{{- end }}
```


