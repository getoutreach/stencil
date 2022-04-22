{{- define "command" }}
{{- /* We should be able to access the root values by default if we had no args passed via ApplyTemplate */}}
{{- .Config.Name }}
{{- end }}
{{- stencil.ApplyTemplate "command" }}