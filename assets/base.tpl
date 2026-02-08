{{define "base"}}
{{- $ctx := . -}}

{{ range .OldText }}
<details><summary>History item...</summary>
{{ range $line := splitLines . }}
> {{ $line -}}
{{- end }}
</details>

{{ end }}

{{- template "content" $ctx -}}
{{- end }}
