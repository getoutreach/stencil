---
title: Contribute to Stencil Development
linktitle: Development
description: Stencil relies heavily on contributions from the open source community.
date: 2022-05-02
publishdate: 2022-05-02
lastmod: 2022-05-02
categories: [contribute]
keywords: [dev,open source]
authors: [jaredallard,digitalcraftsmen]
menu:
  docs:
    parent: "contribute"
    weight: 10
weight: 10
sections_weight: 10
draft: false
toc: true
---

## Introduction

stencil is an open-source project and lives by the work of its [contributors][]. There are plenty of [open issues][issues], and we need your help to make stencil even more awesome. You don't need to be a Go guru to contribute to the project's development.

## Assumptions

This contribution guide takes a step-by-step approach in hopes of helping newcomers. Therefore, we only assume the following:

* You are new to Git or open-source projects in general
* You are a fan of Stencil and enthusiastic about contributing to the project

{{% note "Additional Questions?" %}}
If you're struggling at any point in this contribution guide, reach out to the Stencil community in [Stencil's Discussion forum](forums)
{{% /note %}}

## Install Go

The installation of Go should take only a few minutes. You have more than one option to get Go up and running on your machine.

If you are having trouble following the installation guides for Go, check out [Go Bootcamp, which contains setups for every platform][gobootcamp] or reach out to the Stencil community in the [Stencil Discussion Forums][forums].

### Install Go From Source

[Download the latest stable version of Go][godl] and follow the official [Go installation guide][goinstall].

Once you're finished installing Go, let's confirm everything is working correctly. Open a terminal---or command line under Windows--and type the following:

```
go version
```

You should see something similar to the following written to the console. Note that the version here reflects the most recent version of Go as of the last update for this page:

```
go version go1.18.1 darwin/arm64
```

### Install Go with Homebrew

If you are a MacOS user and have [Homebrew](https://brew.sh/) installed on your machine, installing Go is as simple as the following command:

```bash
brew install go
```

## Create a GitHub Account

If you're going to contribute code, you'll need to have an account on GitHub. Go to [www.github.com/join](https://github.com/join) and set up a personal account.

## Install Git on Your System

You will need to have Git installed on your computer to contribute to Stencil development. Teaching Git is outside the scope of the Stencil docs, but if you're looking for an excellent reference to learn the basics of Git, we recommend the [Git book][gitbook] if you are not sure where to begin. We will include short explanations of the Git commands in this document.

Git is a [version control system](https://en.wikipedia.org/wiki/Version_control) to track the changes of source code. Stencil depends on smaller third-party packages that are used to extend the functionality. We use them because we don't want to reinvent the wheel.

Go ships with a sub-command called `get` that will download these packages for us when we setup our working environment. The source code of the packages is tracked with Git. `get` will interact with the Git servers of the package hosters in order to fetch all dependencies.

Move back to the terminal and check if Git is already installed. Type in `git version` and press enter. You can skip the rest of this section if the command returned a version number. Otherwise [download](https://git-scm.com/downloads) the latest version of Git and follow this [installation guide](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).

Finally, check again with `git version` if Git was installed successfully.

### Git Graphical Front Ends

There are several [GUI clients](https://git-scm.com/downloads/guis) that help you to operate Git. Not all are available for all operating systems and maybe differ in their usage. Because of this we will document how to use the command line, since the commands are the same everywhere.

## Set up your working copy

You set up the working copy of the repository locally on your computer. Your local copy of the files is what you'll edit, compile, and end up pushing back to GitHub. The main steps are cloning the repository and creating your fork as a remote.

### Clone the repository

You should now copy the Stencil repository down to your computer. You'll hear this called "clone the repo". GitHub's [help pages](https://help.github.com/articles/cloning-a-repository/) give us a short explanation:

> When you create a repository on GitHub, it exists as a remote repository. You can create a local clone of your repository on your computer and sync between the two locations.

We're going to clone the [stencil repository](github). That seems counter-intuitive, since you won't have commit rights on it. But it's required for the Go workflow. You'll work on a copy of the remote and push your changes to your own repository on GitHub.

So, let's make a new directory and clone that remote repository:

```
git clone https://github.com/getoutreach/stencil
cd stencil/
```

And then, install dependencies of Stencil by running the following in the cloned directory:

```bash
# This runs `go mod download` under the hood.
make dep
```

### Fork the repository

If you're not familiar with this term, GitHub's [help pages](https://help.github.com/articles/fork-a-repo/) provide again a simple explanation:

> A fork is a copy of a repository. Forking a repository allows you to freely experiment with changes without affecting the original project.

#### Fork using the gh CLI

Since we installed gh earlier, it's super easy to fork this repository.

```
gh repo fork
```
<!-- TODO: Add snippets -->

#### Trust, but verify

Let's check if everything went right by listing all known remotes:

```
git remote -v
```

The output should look similar:

```
jaredallard  git@github.com:jaredallard/stencil (fetch)
jaredallard  git@github.com:jaredallard/stencil (push)
origin  git@github.com:getoutreach/stencil (fetch)
origin  git@github.com:getoutreach/stencil (push)
```

## The Stencil Git Contribution Workflow

### Create a new branch

You should never develop against the "main" branch. The development team will not accept a pull request against that branch. Instead, create a descriptive named branch and work on it.

First, you should always pull the latest changes from the main repository:

```
git checkout main
git pull
```

Now we can create a new branch for your additions:

```
git checkout -b <BRANCH-NAME>
```

You can check on which branch you are with `git branch`. You should see a list of all local branches. The current branch is indicated with a little asterisk.

### Contribute to Documentation

Perhaps you want to start contributing to the stencil docs. If so, you can ignore most of the following steps and focus on the `/docs` directory within your newly cloned repository. You can change directories into the Stencil docs using `cd docs`.

You can render these docs by running `hugo server`. Browse the documentation by entering [http://localhost:1313](http://localhost:1313) in the address bar of your browser. The server automatically updates the page whenever you change content.

We have developed a [separate stencil documentation contribution guide][docscontrib] for more information on how the stencil docs are built, organized, and improved by the generosity of people like you.

### Build Stencil

While making changes in the codebase it's a good idea to build the binary to test them:

```
make
```

This command generates the binary file at `./bin/stencil`


### Test 
Sometimes changes on the codebase can cause unintended side effects. Or they don't work as expected. Most functions have their own test cases. You can find them in files ending with `_test.go`.

Make sure the command `make test` passes.

### Formatting 

The Go code styleguide maybe is opinionated but it ensures that the codebase looks the same, regardless who wrote the code. Go comes with its own formatting tool. Let's apply the styleguide to our additions:

```
make fmt
```

Once you made your additions commit your changes. Make sure that you follow our [code contribution guidelines](https://github.com/getoutreach/stencil/blob/main/CONTRIBUTING.md):

```
# Add all changed files
git add --all
git commit --message "YOUR COMMIT MESSAGE"
```

The commit message should describe what the commit does (e.g. add feature XYZ), not how it is done.

### Modify commits

You noticed some commit messages don't fulfill the code contribution guidelines or you just forget something to add some files? No problem. Git provides the necessary tools to fix such problems. The next two methods cover all common cases.

If you are unsure what a command does leave the commit as it is. We can fix your commits later in the pull request.

#### Modify the last commit

Let's say you want to modify the last commit message. Run the following command and replace the current message:

```
git commit --amend -m "YOUR NEW COMMIT MESSAGE"
```

Take a look at the commit log to see the change:

```
git log
# Exit with q
```

After making the last commit you may have forgot something. There is no need to create a new commit. Just add the latest changes and merge them into the intended commit:

```
git add --all
git commit --amend
```

## Open a pull request

We made a lot of progress. Good work. In this step we finally open a pull request to submit our additions. Using the gh cli we can do so easily.

Run the following:

```bash
# This command takes care of `git push` and `git push --set-upstream origin <BRANCH-NAME>` for you.
gh pr create
```

<!-- TODO: Snippet -->

### Automatic builds

We use a CircleCI workflow to build and test. This is a matrix build across combinations of operating system (masOS, Windows, and Ubuntu) and Go versions. The workflow is triggered by the submission of a pull request. This workflow will be trigerred by a maintainer for security reasons.

## Where to start?

Thank you for reading through this contribution guide. Hopefully, we will see you again soon on GitHub. There are plenty of [open issues][issues] for you to help with.

Feel free to [open an issue][newissue] if you think you found a bug or you have a new idea to improve stencil. We are happy to hear from you.

## Additional References for Learning Git and Go

* [Codecademy's Free "Learn Git" Course][codecademy] (Free)
* [Code School and GitHub's "Try Git" Tutorial][trygit] (Free)
* [The Git Book][gitbook] (Free)
* [Go Bootcamp][gobootcamp]


[codecademy]: https://www.codecademy.com/learn/learn-git
[contributors]: https://github.com/getoutreach/stencil/graphs/contributors
[docscontrib]: /contribute/documentation/
[forums]: https://github.com/getoutreach/stencil/discussions
[gitbook]: https://git-scm.com/
[github]: https://github.com/getoutreach/stencil
[gobootcamp]: https://www.golangbootcamp.com/book/get_setup
[godl]: https://golang.org/dl/
[goinstall]: https://golang.org/doc/install
[gvm]: https://github.com/moovweb/gvm
[issues]: https://github.com/getoutreach/stencil/issues
[newissue]: https://github.com/getoutreach/stencil/issues/new
[releases]: /getting-started/
[setupgopath]: https://golang.org/doc/code.html#Workspaces
[trygit]: https://try.github.io/levels/1/challenges/1
