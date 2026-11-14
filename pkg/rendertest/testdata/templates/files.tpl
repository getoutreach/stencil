{{- file.Create "kept.txt" 0o600 now }}{{ file.SetContents "kept" }}
{{- file.Create "skipped.txt" 0o644 now }}{{ file.Skip "not needed" }}
{{- file.Create "deleted.txt" 0o644 now }}{{ file.Delete }}
