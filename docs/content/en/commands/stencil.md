---
title: stencil
linktitle: stencil
description: a smart templating engine for service development
date: 2022-05-04
categories: [commands]
menu:
  docs:
    parent: "commands"
---

## stencil

```bash
NAME:
   stencil - A new cli application

USAGE:
   stencil [global options] command [command options]

VERSION:
   v1.43.0

DESCRIPTION:
   a smart templating engine for service development

COMMANDS:
   describe  
   create    
   docs      
   module    
   updater   Commands for interacting with the built-in updater
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --concurrent-resolvers value, -c value  Number of concurrent resolvers to use when resolving modules (default: 5)
   --dry-run, --dryrun                     Don't write files to disk (default: false)
   --frozen-lockfile                       Use versions from the lockfile instead of the latest (default: false)
   --use-prerelease                        Use prerelease versions of stencil modules (default: false)
   --allow-major-version-upgrades          Allow major version upgrades without confirmation (default: false)
   --debug, -d                             Enables debug logging for version resolution, template render, and other useful information (default: false)
   --skip-update                           Skips the updater check (default: false)
   --force-update-check                    Force checking for an update (default: false)
   --help, -h                              show help
   --version, -v                           print the version

```
