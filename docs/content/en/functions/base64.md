---
title: base64
description: "`base64Encode` and `base64Decode` let you easily decode content with a base64 encoding and vice versa through pipes."
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
keywords: []
relatedfuncs: []
signature: ["b64dec INPUT", "b64enc INPUT"]
workson: []
deprecated: false
draft: false
aliases: []
---

An example:

{{< code file="base64-input.html" >}}
<p>Hello world = {{ "Hello world" | b64enc }}</p>
<p>SGVsbG8gd29ybGQ = {{ "SGVsbG8gd29ybGQ=" | b64dec }}</p>
{{< /code >}}

{{< output file="base64-output.html" >}}
<p>Hello world = SGVsbG8gd29ybGQ=</p>
<p>SGVsbG8gd29ybGQ = Hello world</p>
{{< /output >}}

You can also pass other data types as arguments to the template function which tries to convert them. The following will convert *42* from an integer to a string because both `b64enc` and `b64dec` always return a string.

```
{{ 42 | b64enc | b64dec }}
=> "42" rather than 42
```