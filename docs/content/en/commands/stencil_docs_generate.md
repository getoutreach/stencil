---
title: stencil docs generate
linktitle: stencil docs generate
description: Generates documentation for the current stencil module
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil docs generate

```bash
NAME:
   stencil docs generate - Generate documentation

USAGE:
   stencil docs generate [options]

DESCRIPTION:
   Generates documentation for the current stencil module

OPTIONS:
   --help, -h  show help

GLOBAL OPTIONS:
   --concurrent-resolvers string, -c string  Number of concurrent resolvers to use when resolving modules (default: 5)
   --dry-run, --dryrun                       Don't write files to disk
   --frozen-lockfile                         Use versions from the lockfile instead of the latest
   --use-prerelease                          Use prerelease versions of stencil modules
   --allow-major-version-upgrades            Allow major version upgrades without confirmation
   --debug, -d                               Enables debug logging for version resolution, template render, and other useful information
   --skip-update                             Skips the updater check
   --force-update-check                      Force checking for an update

```
