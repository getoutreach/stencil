# Module Hooks

Module hooks enable modules to write to sections inside of each other, this allows modules to "hook" into areas;
such as: import slices, glob paths, etc.

## Basic Hook

Let's take a scenario where module A wants to put content into a list that Module B generates so that it can add content to some deployment manifests.

Module B defines a jsonnet section like so:

```tpl
local mixins = [
{{- range := stencil.GetModuleHook "deploymentMixins" }}
'./mixins/{{ . }}',
{{- end }}
];
```

Module A can hook into that section by doing the following:

```tpl
// Don't actually generate this file
{{- file.Skip "passes arguments to module b" }}
{{- stencil.AddToModuleHook "module-b" (list "my-deployment.jsonnet") }}
```

When these modules are both included and then rendered, this would result in the earlier jsonnet file
looking like so:

```jsonnet
local mixins = [
  './mixins/my-deployment.jsonnet',
];
```
