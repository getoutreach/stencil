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
   v1.24.0

DESCRIPTION:
   a smart templating engine for service development

COMMANDS:
   describe  
   create    
   docs      
   updater   Commands for interacting with the built-in updater
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --dry-run, --dryrun             Don't write files to disk (default: false)
   --frozen-lockfile               Use versions from the lockfile instead of the latest (default: false)
   --use-prerelease                Use prerelease versions of stencil modules (default: false)
   --allow-major-version-upgrades  Allow major version upgrades without confirmation (default: false)
   --skip-update                   skips the updater check (default: false)
   --debug                         enables debug logging for all components that use logrus (default: false)
   --enable-prereleases            Enable considering pre-releases when checking for updates (default: false)
   --force-update-check            Force checking for an update (default: false)
   --help, -h                      show help (default: false)
   --version, -v                   print the version (default: false)

```
