# Templates

A template is a `<filename>.<extension>.tpl` in a stencil module. 

## Render Behavior

**Location**: By default these write to the same file as the template with `strings.TrimSuffix(name, ".tpl")` in the directory that stencil is being ran in. This can be changed with `file.SetPath`.

**Mode**: By default the mode of the file is taken from template's mode, this can be changed with `file.SetMode`.

## File/Stencil Functions

The `file` and `stencil` functions are scoped to a given template/module (depending on the context) that's currently
being rendered by stencil.

A complete list of all functions available to a template can be found in pkg.go.dev:

 * [`file`](https://pkg.go.dev/github.com/getoutreach/stencil@v1.5.0/internal/codegen#TplFile) 
 * [`stencil`](https://pkg.go.dev/github.com/getoutreach/stencil@v1.5.0/internal/codegen#TplStencil)


## Creating Multiple Files From a Single Template

Templates are 1:1, template to file, ratio when ran by default. This can be changed by calling `file.Create` in a loop:

```tpl
{{- range (list "a" "b" "c") }}
{{- file.Create (printf "%s.txt" .) 0600 now }}
{{- file.SetContents "hello" }}
{{- end }}
```

## Executing Sub-Templates

Go templates have the notion of defining templates within a template. Stencil doesn't treat these as unique files, by default.

Instead, you can call an embedded template via `stencil.ApplyTemplate`, below is an example:

```tpl
{{- define "hello-world" }}
hello, world!
{{- end }}

{{- stencil.ApplyTemplate "hello-world" }}
```

**Note**: `file` and `stencil`'s context will be passed to the sub-template, so if a sub-template creates files they will be
marked as being owned from the caller's template context.
