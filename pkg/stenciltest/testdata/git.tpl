{{- if ne .Git.DefaultBranch "main" }}{{ fail "expected .Git.DefaultBranch to be main" }}{{ end }}
{{- if ne .Git.Ref "refs/heads/main" }}{{ fail "expected .Git.Ref to be refs/heads/main" }}{{ end }}
{{- if ne .Git.Commit "0000000000000000000000000000000000000000" }}{{ fail "expected .Git.Commit to be all zeros" }}{{ end }}
{{- if .Git.Dirty }}{{ fail "expected .Git.Dirty to be false" }}{{ end }}
