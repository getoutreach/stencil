---
title: file.Static
linktitle: file.Static
description: >
  Static marks the current file as static
date: 2022-05-18
categories: [functions]
menu:
  docs:
    parent: "functions"
---


Marking a file is equivalent to calling file\.Skip\, but instead file\.Skip is only called if the file already exists\. This is useful for files you want to generate but only once\. It's generally recommended that you do not do this as it limits your ability to change the file in the future\.


```go-text-template
{{ $_ := file.Static }}
```


