# stencil

Stencil is a go-template powered micro-service lifecycle manager.

## How to Use

Download the [latest release](/releases) and extract it.

```bash
# Create a directory for your new service, or run in an existing stencil service dir
$ cd my-new-service
$ ./stencil
```

Profit.

## Creating a new service

**TODO**

For now you can simply create a [`service.yaml`](ADD LINK AFTER OSS) and add a module to the list.

## Writing Templates

Templates are written via [go-template](https://pkg.go.dev/text/template) syntax. Simply create a new module repository with a [`manifest.yaml`](ADD LINK AFTER OSS) and create a `.tpl` file to have it be automatically included / rendered. By default a file is written to the same place as the name it has in the template repository. This can be changed with the `file.SetPath` function.

### Functions

All available template functions are able to be found on [pkg.dev](ADD LINK AFTER OSS).

# License

Apache-2.0
