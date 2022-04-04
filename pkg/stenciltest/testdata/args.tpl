{{ if ne (stencil.Arg "hello") "world" }}
{{ fail "expected .hello to be 'world' "}}
{{ end }}
