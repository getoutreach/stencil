---
title: stencil updater use
linktitle: stencil updater use
description: 
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil updater use

```bash
NAME:
   stencil updater use - Use a specific version of the application

USAGE:
   stencil updater use [options]

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
