{{define "content" -}}
BeepBoop: new job started
Log tracking url: {{.MyDSN}}/api/v1/job/logs?host={{.JobHost}}&job_id={{.JobResponse.JobId}}
{{- end}}
