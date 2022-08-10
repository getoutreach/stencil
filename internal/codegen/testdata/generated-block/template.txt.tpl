{{- range $_, $block := (list "a" "b" "c") }}
###Block({{ $block }})
{{ file.Block $block }}
###EndBlock({{ $block }})
{{- end }}
