---
title: stencil describe
linktitle: stencil describe
description: Print information about a known file rendered by a template
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil describe

```bash
NAME:
   stencil describe

USAGE:
   stencil describe [options]

DESCRIPTION:
   Print information about a known file rendered by a template

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
