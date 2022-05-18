---
title: Stencil Modules
linktitle: Stencil Modules Overview
description: How to use Stencil Modules.
date: 2022-05-04
publishdate:  2022-05-04
menu:
  docs:
    parent: "reference"
    weight: 1
categories: [modules]
keywords: [modules]
toc: true
---

**Stencil Modules** are the core building blocks in Stencil.

Modules are used to create reusable grouping of templates and native extensions to be used by stencil.

Stencil Modules are powered by Go Modules and must be within a git repository. For more information about Go Modules, see:

- [https://github.com/golang/go/wiki/Modules](https://github.com/golang/go/wiki/Modules)
- [https://blog.golang.org/using-go-modules](https://blog.golang.org/using-go-modules)

## Types of Modules

There are two types of module usable by stencil, a module and a native extension. A module may, itself, be only one of these types of modules.

### Module

A module consists of templates in a `templates/` directory in the root of the repository. Modules are written in the [go template syntax](https://pkg.go.dev/text/template) with added functions and variables accessible to them at runtime. For more information about the module type see [the basic module documentation](/stencil/reference/template-module).

### Native Extensions

Native extensions are modules that run binary code, and generally are written in Go but may be written in any language that can implement a `net/rpc` interface. Native extensions are accessible via the `extensions.Call "<importPath>.<functionName>"` method. For more information about the native extension module type see [the native extension module documentation]().

## Creating a Module

For information on how to create a module see the [getting started](/stencil/getting-started/) documentation.

## More Information

For more technical documentation on modules, see the below links.