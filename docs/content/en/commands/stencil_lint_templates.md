---
title: stencil lint templates
linktitle: stencil lint templates
description: Validate template files (defaults to ./templates/**/*.tpl). Use '-' to read a single template from stdin.
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil lint templates

```bash
NAME:
   stencil lint templates - Validate Stencil templates' block correctness without rendering

USAGE:
   stencil lint templates [options] [files...]

DESCRIPTION:
   Validate template files (defaults to ./templates/**/*.tpl). Use '-' to read a single template from stdin.

OPTIONS:
   --warnings-as-errors  treat warnings as errors (fail on any finding)
   --help, -h            show help

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
