{{define "content" -}}
BeepBoop: new job started
Log tracking url: {{.MyDSN}}/job/logs?host={{.JobHost}}&job_id={{.JobResponse.JobId}}
{{- end}}
