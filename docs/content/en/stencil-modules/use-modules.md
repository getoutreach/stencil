---
title: Use Stencil Modules
linktitle: Use Stencil Modules
description: How to use Stencil Modules to build and manage your site.
date: 2022-05-02
categories: [stencil modules]
keywords: [install, themes, source, organization, directories, usage, modules]
menu:
  docs:
    parent: "modules"
    weight: 20
weight: 20
sections_weight: 20
draft: false
aliases: [/themes/usage/, /themes/installing/, /installing-and-using-themes/]
toc: true
---

## Prerequisite

{{< gomodules-info >}}

## Initialize a New Module

Use `stencil create templaterepository` to initialize a new Stencil Module. If it fails to guess the module path, you must provide it as an argument, e.g.:

```bash
stencil create templaterepository github.com/myorg/repo
```

<!-- TODO: How to use a module>
