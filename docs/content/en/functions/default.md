---
title: default
description: Allows setting a default value that can be returned if a first value is not set.
qref: "Returns a default value if a value is not set when checked."
date: 2022-05-02
keywords: [defaults]
categories: [functions]
menu:
  docs:
    parent: "functions"
toc:
signature: ["default DEFAULT INPUT"]
deprecated: false
draft: false
aliases: []
---

`default` checks whether a given value is set and returns a default value if it is not. *Set* in this context means different things depending on the data type:

* non-zero for numeric types and times
* non-zero length for strings, arrays, slices, and maps
* any boolean or struct value
* non-nil for any other types

`default` can be written in more than one way:

```
{{ index (stencil.Arg "array") 0 | default "abc" }}
{{ default "abc" (index (stencil.Arg "array)) }}
```

Both of the above `default` function calls return `abc`.

A `default` value, however, does not need to be hard coded like the previous example. The `default` value can be a variable:

{{< code file="arg-as-default-value.html" nocopy="true" >}}
{{ $old := stencil.Arg "oldArg" }}
{{ stencil.Arg "newArg" | default $old }}
{{< /code >}}

The following have equivalent return values but are far less terse. This demonstrates the utility of `default`:

Using `if`:

{{< code file="if-instead-of-default.html" nocopy="true" >}}
<title>{{if stencil.Arg "oldArg" }}{{stencil.Arg "oldArg}}{{else}}{{stencil.Arg "newArg}}{{end}}</title>
=> Sane Defaults
{{< /code >}}
