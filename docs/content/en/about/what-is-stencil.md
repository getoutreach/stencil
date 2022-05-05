---
title: What is Stencil?
linktitle: What is Stencil?
description: A smart templating engine for service development
date: 2022-05-02
publishdate: 2022-05-02
lastmod: 2022-05-02
layout: single
menu:
  docs:
    parent: "about"
    weight: 1
aliases: [/overview/introduction/]
toc: true
---

Stencil is a tool to eliminate run-once boilerplate code generally created by tools like `create-react-app` and eliminate the decade old practice of copying and pasting boilerplate from stackoverflow and other websites.

Instead you can write modules for your boilerplate and other "components" of an application, library, or etc and use them in your application/library/whatever else you dare to create.

Stencil enables this through a few different components:

* [Modules](/stencil/modules/) - The major selling point of stencil, allows you to create module groups of Go templates and other dependencies.
* [Native Extensions](/stencil/modules/native-extensions) - Write code in any language and consume it within your templates.
* [Module Hooks](/stencil/modules/template-module#module-hooks) - Allows modules to write to templates a module owns in a predictable strongly typed fashion, enabling various extension points.
* [Easy Debuggability](/stencil/commands/describe/) - Using the `stencil describe <path>` command you're able to see which module created which file when and where, removing the complexity of who created what.

Curious? Dig into our [Getting Started](/stencil/getting-started/) docs to start creating!