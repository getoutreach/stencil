{{- $_ := stencil.AddToModuleHook "github.com/example/dep" "produced" (list "from-module") }}
{{- range stencil.GetModuleHook "consumed" }}consumed={{ . }}
{{ end -}}
