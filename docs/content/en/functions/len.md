---
title: len
linktitle: len
description: Returns the length of a variable according to its type.
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
keywords: []
signature: ["len INPUT"]
workson: [lists, taxonomies, terms]
deprecated: false
toc: false
---

`len` is a built-in function in Go that returns the length of a variable according to its type. From the Go documentation:

> Array: the number of elements in v.
>
> Pointer to array: the number of elements in \*v (even if v is nil).
>
> Slice, or map: the number of elements in v; if v is nil, len(v) is zero.
>
> String: the number of bytes in v.
>
> Channel: the number of elements queued (unread) in the channel buffer; if v is nil, len(v) is zero.
