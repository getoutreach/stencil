---
title: Install Stencil
linktitle: Install Stencil
description: Install Stencil on macOS, Windows, Linux, OpenBSD, FreeBSD, and on any machine where the Go compiler tool chain can run.
date: 2022-05-02
categories: [getting started, fundamentals]
authors: ["Michael Henderson", "Jared Allard"]
keywords: [install, pc, windows, linux, macos, binary, tarball]
menu:
  docs:
    parent: "getting-started"
    weight: 5
aliases: [/install/]
toc: true
---

{{% note %}}
There is lots of talk about "Stencil being written in Go", but you don't need to install Go to enjoy Stencil. Just grab a [precompiled](https://github.com/getoutreach/stencil/releases/latest) binary!
{{% /note %}}

Stencil is written in [Go](https://golang.org/) with support for multiple platforms. The latest release can be found at [Stencil Releases][releases].

Stencil currently provides pre-built binaries for the following:

- macOS (Darwin) for x64, i386, and ARM architectures
- Windows
- Linux

Stencil may also be compiled from source wherever the Go toolchain can run; e.g., on other operating systems such as DragonFly BSD, OpenBSD, Plan&nbsp;9, Solaris, and others. See <https://golang.org/doc/install/source> for the full set of supported combinations of target operating systems and compilation architectures.

## Quick Install

### Homebrew (macOS)

We have a brew formula for Stencil. It is recommended to install Stencil via Homebrew on macOS.

```bash
brew install stencil
```

### Binary (Cross-platform)

Download the appropriate version for your platform from [Stencil Releases][releases]. Once downloaded, the binary can be run from anywhere. You don't need to install it into a global location. This works well for shared hosts and other systems where you don't have a privileged account.

Ideally, you should install it somewhere in your `PATH` for easy use. `/usr/local/bin` is the most probable location.

### Source

#### Prerequisite Tools

- [Git][installgit]
- [Go (at least Go 1.18)](https://golang.org/dl/)

#### Fetch from GitHub

{{< code file="from-gh.sh" >}}
git clone https://github.com/getoutreach/stencil.git
cd stencil
make
cp ./bin/stencil "$(go env GOPATH)/bin/stencil"
{{< /code >}}

## Upgrade Stencil

Upgrading Stencil is as easy as downloading and replacing the executable you’ve placed in your `PATH`.

## Next Steps

Now that you've installed Stencil, read the [Quick Start guide][quickstart] and explore the rest of the documentation. If you have questions, ask the Stencil community directly by visiting the [Stencil discussion Forum][forum].

[forum]: https://github.com/getoutreach/stencil/discussions
[installgit]: https://git-scm.com/
[installgo]: https://golang.org/dl/
[quickstart]: /stencil/getting-started/quick-start/
[releases]: https://github.com/getoutreach/stencil/releases
