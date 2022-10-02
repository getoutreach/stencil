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
   stencil [global options] command [command options] [arguments...]

VERSION:
   v1.28.0

DESCRIPTION:
   a smart templating engine for service development

COMMANDS:
   describe  
   create    
   docs      
   updater   Commands for interacting with the built-in updater
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --allow-major-version-upgrades  Allow major version upgrades without confirmation (default: false)
   --debug, -d                     Enables debug logging for version resolution, template render, and other useful information (default: false)
   --dry-run, --dryrun             Don't write files to disk (default: false)
   --force-update-check            Force checking for an update (default: false)
   --frozen-lockfile               Use versions from the lockfile instead of the latest (default: false)
   --help, -h                      show help (default: false)
   --skip-update                   skips the updater check (default: false)
   --use-prerelease                Use prerelease versions of stencil modules (default: false)
   --version, -v                   print the version (default: false)
   

```
