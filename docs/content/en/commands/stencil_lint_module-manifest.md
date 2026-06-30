---
title: stencil lint module-manifest
linktitle: stencil lint module-manifest
description: Validate a single template repository manifest (manifest.yaml; defaults to ./manifest.yaml). Use '-' to read from stdin.
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil lint module-manifest

```bash
NAME:
   stencil lint module-manifest - Validate a module's manifest.yaml without resolving dependencies

USAGE:
   stencil lint module-manifest [options] [path]

DESCRIPTION:
   Validate a single template repository manifest (manifest.yaml; defaults to ./manifest.yaml). Use '-' to read from stdin.

OPTIONS:
   --warnings-as-errors  treat warnings as errors (fail on any finding)
   --fix                 automatically fix safe deprecations in place, re-encoding the manifest at 2-space indent when a fix is applied (re-lints after fixing)
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
