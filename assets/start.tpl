{{define "content" -}}
BeepBoop: new job started
Log tracking url: {{.MyDSN}}/job/logs?host={{.JobHost}}&job_id={{.JobResponse.JobId}}
{{ if gt (len .CustomFlags) 0 }}
Detected Docker Environment:
{{ "\n" -}}
    {{- range $value := .CustomFlags }}
    {{- " " -}}{{- $value -}}{{ " " -}}
	{{ end -}}
{{ end }}

{{- end}}
