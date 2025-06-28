{{define "content"}}
#### BeepBoop: you called {{.HelpCommand}}
##### Supported commands:
{{- range .SupportedCommands }}
- {{ . }};
{{- end }}

{{ with .StartHelp }}
@my_tag {{ .StartCommand }} HOST WORKFLOW_PATH
#### Ex: `@my_tag {{ .StartCommand }} h200 .github/workflows/push_check.yaml`
{{ end -}}

#### Supported hosts:
{{- range $key, $value := .Hosts.Hosts }}
#### ======Name=========
#### {{ $key }}
===================
{{- end }}
{{- end }}