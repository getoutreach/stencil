{{- $str := "helloWorld" }}
{{- $resp := extensions.Call "inproc.echo" $str }}
{{- eq $resp $str | toString }}