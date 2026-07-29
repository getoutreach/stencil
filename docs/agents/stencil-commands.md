# Stencil Commands

Stencil commands are exposed as `mise` tasks. To list all available stencil tasks:

```bash
mise tasks | grep stencil
```

## Common tasks

| Task | Description |
|---|---|
| `stencil` | Run Stencil without updating module versions |
| `stencil:upgrade` | Run Stencil and update module versions if available |

## Usage

```bash
mise run stencil
mise run stencil:upgrade
```

For full task details:

```bash
mise tasks info <task-name>
```
