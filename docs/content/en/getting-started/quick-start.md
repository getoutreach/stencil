---
title: Quick Start
linktitle: Quick Start
description: Download stencil and use a template repository
date: 2022-05-02
publishdate: 2022-05-02
categories: [getting started]
keywords: [quick start, usage]
authors: [jaredallard]
menu:
  docs:
    parent: "getting-started"
    weight: 10
weight: 10
sections_weight: 10
draft: false
aliases: [/quickstart/, /overview/quickstart/]
toc: true
---

{{% note %}}
This quick start uses `macOS` in the examples. For instructions about how to install Stencil on other operating systems, see [install](/getting-started/installing).

It is required to have [Git installed](https://git-scm.com/downloads) to run this tutorial.
{{% /note %}}

## Step 1: Install Stencil

If you don't already have stencil installed, follow the [documentation to install it]({{< relref "installing">}}).

## Step 2: Create a `service.yaml`

First start by creating a new directory for your application, this should generally match whatever the name for your application is going to be. We'll go with `helloworld` here.

{{< code file="mkdir.sh" copy=true >}}
mkdir helloworld
{{< /code >}}

A [`service.yaml`](/stencil/reference/service.yaml/) is integral to running stencil. It defines the modules you're consuming and the arguments to pass to them.

Start with creating a basic `service.yaml`:

{{< code file="service.yaml" copy=true >}}
name: helloworld
arguments: {}

# Below is a list of modules to use for this application

# It should follow the following format:

# - name: <moduleImportPath>

# version: "optional-version-to-pin-to"

modules: []
{{< /code >}}

Now run `stencil`, you should have... nothing! That's expected because we didn't define any modules yet.

{{< code file="ls.sh" >}}
helloworld ❯ stencil

INFO[0000] stencil v1.14.2
INFO[0002] Fetching dependencies
INFO[0002] Loading native extensions
INFO[0002] Rendering templates
INFO[0002] Writing template(s) to disk
INFO[0002] Running post-run command(s)

helloworld ❯ ls -alh
drwxr-xr-x 4 jaredallard wheel 128B May 4 20:16 .
drwxr-xr-x 9 jaredallard wheel 288B May 4 20:16 ..
-rw-r--r-- 1 jaredallard wheel 213B May 4 20:16 service.yaml
-rw-r--r-- 1 jaredallard wheel 78B May 4 20:16 stencil.lock
{{< /code >}}

{{% note %}}
You'll notice there's a `stencil.lock` file here.

```yaml
version: v1.14.2
generated: 2022-05-05T03:16:44.903458Z
modules: []
files: []
```

This will keep track of what files were created by stencil and what created them, as well as the last ran version of your modules. This file is very important!

{{% /note %}}

## Step 3: Import a Module

Now that we've created our first stencil application, you're going to want to import a module! Let's import the [`stencil-base`](https://github.com/getoutreach/stencil-base) module. stencil-base includes a bunch of scripts and other build blocks for a service. Let's take a look at it's `manifest.yaml` to see what arguments are required.

{{< code file="manifest.yaml" >}}
name: github.com/getoutreach/stencil-base
...
arguments:
description:
required: true
type: string
description: The purpose of this repository.
{{< /code >}}

We can see that `description` is a required argument, so let's add it! Modify the `service.yaml` to set `arguments.description` to `"My awesome service!"`

{{< code file="service.yaml" >}}
name: helloworld
arguments:
description: "My awesome service!"
modules:

- name: github.com/getoutreach/stencil-base
  {{< /code >}}

Now if we run stencil we'll see that we have some files!

```bash
helloworld ❯ stencil
INFO[0000] stencil v1.14.2
INFO[0000] Fetching dependencies
INFO[0001]  -> github.com/getoutreach/stencil-base v0.2.0
INFO[0001] Loading native extensions
INFO[0001] Rendering templates
INFO[0001] Writing template(s) to disk
INFO[0001]   -> Created .editorconfig
INFO[0001]   -> Created .github/CODEOWNERS
INFO[0001]   -> Created .github/pull_request_template.md
INFO[0001]   -> Created .gitignore
INFO[0001]   -> Created .releaserc.yaml
INFO[0001]   -> Created CONTRIBUTING.md
INFO[0001]   -> Skipped LICENSE
INFO[0001]   -> Created README.md
INFO[0001]   -> Skipped documentation/README.md
INFO[0001]   -> Skipped documentation/SLOs.md
INFO[0001]   -> Skipped documentation/disaster-recovery.md
INFO[0001]   -> Skipped documentation/rollout-plan.md
INFO[0001]   -> Skipped documentation/runbook.md
INFO[0001]   -> Skipped helpers
INFO[0001]   -> Created package.json
INFO[0001]   -> Deleted scripts/bootstrap-lib.sh
INFO[0001]   -> Created scripts/devbase.sh
INFO[0001]   -> Created scripts/shell-wrapper.sh
INFO[0001] Running post-run command(s)
...

helloworld ❯ ls -alh
drwxr-xr-x   4 jaredallard  wheel   128B May  4 20:16 .
drwxr-xr-x   9 jaredallard  wheel   288B May  4 20:16 ..
-rw-r--r--   1 jaredallard  wheel   274B May  4 20:26 .editorconfig
drwxr-xr-x   4 jaredallard  wheel   128B May  4 20:26 .github
-rw-r--r--   1 jaredallard  wheel   795B May  4 20:26 .gitignore
-rw-r--r--   1 jaredallard  wheel   857B May  4 20:26 .releaserc.yaml
-rw-r--r--   1 jaredallard  wheel   417B May  4 20:26 CONTRIBUTING.md
-rw-r--r--   1 jaredallard  wheel   703B May  4 20:26 README.md
-rw-r--r--   1 jaredallard  wheel   474B May  4 20:26 package.json
drwxr-xr-x   4 jaredallard  wheel   128B May  4 20:26 scripts
-rw-r--r--   1 jaredallard  wheel   118B May  4 20:26 service.yaml
-rw-r--r--   1 jaredallard  wheel   2.1K May  4 20:26 stencil.lock
```

# Step 4: Modifying a Block

One of the key features in stencil is the notion of "blocks". Modules can expose a block where they want developers to modify the code. Let's look at the `stencil-base` module to see what blocks are available.

In `README.md` we can see a basic block called "overview"

{{< code file="README.md" copy=true >}}

# helloworld

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/getoutreach/helloworld)
[![Generated via Stencil](https://img.shields.io/badge/Outreach-Bootstrap-%235951ff)](https://github.com/getoutreach/stencil)
[![Coverage Status](https://coveralls.io/repos/github/getoutreach/helloworld/badge.svg?branch=main)](https://coveralls.io/github/getoutreach/helloworld?branch=)

My awesome service!

## Contributing

Please read the [CONTRIBUTING.md](CONTRIBUTING.md) document for guidelines on developing and contributing changes.

## High-level Overview

<!--- Block(overview) -->

<!--- EndBlock(overview) -->

{{< /code >}}

Let's add some content in two places. One inside the block, one outside.

{{< code file="README.additions.md">}}
...

## High-level Overview

hello, world!

<!--- Block(overview) -->

hello, world!

<!--- EndBlock(overview) -->

{{< /code >}}

If we re-run stencil, notice how the contents of `README.md` have changed.

{{< code file="README.md" copy=true >}}

# helloworld

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/getoutreach/helloworld)
[![Generated via Stencil](https://img.shields.io/badge/Outreach-Bootstrap-%235951ff)](https://github.com/getoutreach/stencil)
[![Coverage Status](https://coveralls.io/repos/github/getoutreach/helloworld/badge.svg?branch=main)](https://coveralls.io/github/getoutreach/helloworld?branch=)

My awesome service!

## Contributing

Please read the [CONTRIBUTING.md](CONTRIBUTING.md) document for guidelines on developing and contributing changes.

## High-level Overview

<!--- Block(overview) -->

hello, world!

<!--- EndBlock(overview) -->

{{< /code >}}

The contents of `README.md` have changed, but the contents within the block have not. This is the power of blocks, modules are able to change the content _around_ a user's content without affecting the user's content. This can be taken even further if a template decides to parse the code within a block at runtime, for example using the ast package to rewrite go code.

## Reflection

In all, we've created a `service.yaml`, added a module to it, ran stencil, and then modified the contents within a block. That's it! We've imported a base module and created some files, all without doing much of anything on our side. Hopefully that shows you the power of stencil.

For more resources be sure to dive into [how to create a module]({{< relref "module-quick-start.md" >}}) to get insight on how to create a module.
