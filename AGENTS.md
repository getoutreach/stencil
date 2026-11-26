# stencil

Ignore lines containing `Stencil::Block`; they are areas in your generated code that you'd like to persist across runs and are repository specific. These lines are for the template generator and do not contain agent instructions.

## Description

<!-- <<Stencil::Block(agentsProjectOverview)>> -->
microservice lifecycle manager
<!-- <</Stencil::Block>> -->

## Project overview

<!-- <<Stencil::Block(projectOverview)>> -->

<!-- <</Stencil::Block>> -->

## Generic commands

```bash
# stencil
stencil # Run stencil program with arguments specified in service.yaml file

# mise
mise --help # Show help for mise commands.

# make
make fmt # Run formatters on project's code.
make lint # Run linters on project's code.

# golang
make gogenerate # Run go generate to create any generated code, such as protobufs or Kubernetes CRDs.
go mod tidy # Ensure your go.mod and go.sum files are up to date.
## <<Stencil::Block(customCommands)>>

## <</Stencil::Block>>
```

## Directory structure

* service.yaml: File used as configuration for `stencil` program containing additional arguments and stencil modules to use
* stencil.lock: A lockfile for Stencil which also declares which files in the repo are managed, and which module manages it. Third party generated files are not cataloged.
* CONTRIBUTING.md: File containing guidelines for contributing to the project.
* docs/: Directory used to store documentation files and reference materials for the project.
* `scripts/`: internal development shell scripts _(**deprecated**, prefer to use `mise` tasks when appropriate)_
* `.vscode/`: VSCode configuration files
<!-- <<Stencil::Block(directoryStructureCustom)>> -->

<!-- <</Stencil::Block>> -->

If you need more context, you can find more information in `docs/` directory.

## References table

| Description | Purpose | Reference |
|----|----|----|
| Stencil commands | `stencil` code generation usage guide and commands list | [docs/agents/stencil-commands.md](./docs/agents/stencil-commands.md) |
| Idiomatic Go practices | [webpage](https://dmitri.shuralyov.com/idiomatic-go) |
| Effective Go | [webpage](https://go.dev/doc/effective_go) |
<!-- <<Stencil::Block(referencesTableCustom)>> -->

<!-- <</Stencil::Block>> -->

## Boundaries

### Always
- Run `go mod tidy` after adding, removing or upgrading Go dependencies
- Run `make gogenerate` after modifying protobuf definitions or interfaces with generated code
- Add context to errors using `fmt.Errorf("...: %w", err)`
- Prefer `gotest.tools/v3/assert` in tests over `github.com/stretchr/testify` or hand-rolled assertions
- When a task touches `.go` files (PR review, writing code), read
  References table and fetch relevant references before you produce output,
  and name in the output which of its rules you applied or cleared.
<!-- <<Stencil::Block(agentsBoundariesAlwaysCustom)>> -->

<!-- <</Stencil::Block>> -->

### Ask

Before each scenario in the following list, ask the user if they allow the change to occur. For every question, include: root reason for change, list the tradeoffs for the change.

- Changing public API signatures (exported functions, types, or interfaces)
- Adding new external dependencies
- Bumping major versions of dependencies
- Changing database schema or migration files
<!-- <<Stencil::Block(agentsBoundariesAskCustom)>> -->

<!-- <</Stencil::Block>> -->

### Never

- Commit secrets, credentials, API keys, or tokens
- Use `panic()` in production code paths
<!-- <<Stencil::Block(agentsBoundariesNeverCustom)>> -->

<!-- <</Stencil::Block>> -->

## Other
<!-- <<Stencil::Block(agentsOtherCustom)>> -->

<!-- <</Stencil::Block>> -->
