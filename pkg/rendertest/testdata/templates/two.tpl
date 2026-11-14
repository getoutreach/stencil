{{- file.Create "one.txt" 0o644 now }}{{ file.SetContents "one\n" }}
{{- file.Create "dir/two.txt" 0o644 now }}{{ file.SetContents "two\n" }}
