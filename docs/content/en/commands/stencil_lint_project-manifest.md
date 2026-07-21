---
title: stencil lint project-manifest
linktitle: stencil lint project-manifest
description: Validate a single project manifest (service.yaml; defaults to ./service.yaml). Use '-' to read from stdin.
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil lint project-manifest

```bash
NAME:
   stencil lint project-manifest - Validate a project's service.yaml without resolving dependencies

USAGE:
   stencil lint project-manifest [options] [path]

DESCRIPTION:
   Validate a single project manifest (service.yaml; defaults to ./service.yaml). Use '-' to read from stdin.

OPTIONS:
   --warnings-as-errors  treat warnings as errors (fail on any finding)
   --offline             skip module resolution; run only offline syntactic checks
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
