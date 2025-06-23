{{define "base"}}

{{- $ctx := . -}}
{{- range .OldText }}
<details><summary>History item...</summary>{{ . }}</details>{{ with $ctx.Sep }}{{ . }}{{ end }}
{{- end }}
{{- template "content" $ctx -}}
{{- end }}
