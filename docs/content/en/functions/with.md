---
title: with
description: Rebinds the context (`.`) within its scope and skips the block if the variable is absent or empty.
date: 2022-05-02
categories: [functions]
menu:
  docs:
    parent: "functions"
keywords: [conditionals]
signature: ["with INPUT"]
---

An alternative way of writing an `if` statement and then referencing the same value is to use `with` instead. `with` rebinds the context (`.`) within its scope and skips the block if the variable is absent, unset or empty.

The set of *empty* values is defined by [the Go templates package](https://golang.org/pkg/text/template/). Empty values include `false`, the number zero, and the empty string.