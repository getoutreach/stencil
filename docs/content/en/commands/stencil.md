---
title: stencil
linktitle: stencil
description: a smart templating engine for service development
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
   stencil [global options] [command [command options]]

VERSION:
   v1.44.0-rc.2

DESCRIPTION:
   a smart templating engine for service development

COMMANDS:
   describe  
   create    
   docs      
   module    
   lint      Validate a Stencil module without resolving dependencies
   updater   Commands for interacting with the built-in updater
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --concurrent-resolvers string, -c string  Number of concurrent resolvers to use when resolving modules (default: 5)
   --dry-run, --dryrun                       Don't write files to disk
   --frozen-lockfile                         Use versions from the lockfile instead of the latest
   --use-prerelease                          Use prerelease versions of stencil modules
   --allow-major-version-upgrades            Allow major version upgrades without confirmation
   --debug, -d                               Enables debug logging for version resolution, template render, and other useful information
   --skip-update                             Skips the updater check
   --force-update-check                      Force checking for an update
   --help, -h                                show help
   --version, -v                             print the version

```
