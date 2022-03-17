{{- define "command" }}
{{- . }}
{{- end }}

# Generate a "<commandName>.go" file for each command in .arguments.commands
{{- range $_, $commandName := (stencil.Arg "commands") }}
{{- file.Create (printf "cmd/%s.go" $commandName) 0644 now  }}
{{- stencil.ApplyTemplate "command" $commandName | file.SetContents }}
{{- end }}
