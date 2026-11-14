{{- define "greeting" }}hello, {{ stencil.Arg "name" }}{{ end }}
{{- stencil.SetGlobal "fromHelper" "global value" }}
{{- file.Skip "helper" }}
