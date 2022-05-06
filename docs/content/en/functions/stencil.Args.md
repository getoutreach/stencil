---
title: stencil.Args
linktitle: stencil.Args
description: >
  Args returns all arguments passed to stencil from the service's manifest. Note: This doesn't set default values and is instead representative of _all_ data passed in its raw form.
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
---

Args returns all arguments passed to stencil from the service's manifest\. Note: This doesn't set default values and is instead representative of \_all\_ data passed in its raw form\.

```go-text-template
{{- (stencil.Args).name }}
```

