---
title: stencil lint
linktitle: stencil lint
description: Validate a Stencil module's manifest without resolving dependencies (template linting follows in DT-4828)
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
   Validate a Stencil module's manifest without resolving dependencies (template linting follows in DT-4828)

COMMANDS:
   module-manifest  Validate a module's manifest.yaml without resolving dependencies

OPTIONS:
   --warnings-as-errors  treat warnings as errors (fail on any finding)
   --fix                 automatically fix safe deprecations in place (re-lints after fixing)
   --help, -h            show help

```
