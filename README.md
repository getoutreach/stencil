
# stencil
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/getoutreach/stencil)
[![Generated via Bootstrap](https://img.shields.io/badge/Outreach-Bootstrap-%235951ff)](https://github.com/getoutreach/bootstrap)

microservice lifecycle manager

## Contributing

Please read the [CONTRIBUTING.md](CONTRIBUTING.md) document for guidelines on developing and contributing changes.

## High-level Overview

<!--- Block(overview) -->
### Creating a new service

**TODO**

For now you can simply create a [`service.yaml`](https://github.com/getoutreach/stencil/blob/main/pkg/configuration/configuration.go#L33) and add a module to the list.

### Writing Templates

Templates are written via [go-template](https://pkg.go.dev/text/template) syntax. Simply create a new module repository with a [`manifest.yaml`](https://github.com/getoutreach/stencil/blob/main/pkg/configuration/configuration.go#L61) and create a `.tpl` file to have it be automatically included / rendered. By default a file is written to the same place as the name it has in the template repository. This can be changed with the `file.SetPath` function.

#### Functions

All available template functions are able to be found on [pkg.go.dev](https://pkg.go.dev/github.com/getoutreach/stencil/pkg/functions).

<!--- EndBlock(overview) -->
