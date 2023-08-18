{{- $mh := stencil.GetModuleHook "coolthing" }}
{{ if $mh }}{{ index $mh 0 }}{{ end }}
