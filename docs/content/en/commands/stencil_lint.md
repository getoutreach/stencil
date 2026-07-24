---
title: stencil lint
linktitle: stencil lint
description: Validate a Stencil module's manifest and templates without resolving dependencies
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil lint

```bash
NAME:
   stencil lint - Validate a Stencil module without resolving dependencies

USAGE:
   stencil lint [command [command options]] [dir]

DESCRIPTION:
   Validate a Stencil module's manifest and templates without resolving dependencies

COMMANDS:
   module-manifest   Validate a module's manifest.yaml without resolving dependencies
   templates         Validate Stencil templates' block correctness without rendering
   project-manifest  Validate a project's service.yaml without resolving dependencies

OPTIONS:
   --warnings-as-errors  treat warnings as errors (fail on any finding)
   --fix                 automatically fix safe deprecations in place (a manifest is re-encoded at 2-space indent when fixed; re-lints after fixing)
   --offline             skip module resolution; run only offline syntactic checks
   --help, -h            show help

```
