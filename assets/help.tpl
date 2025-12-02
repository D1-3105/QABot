{{define "content"}}
#### BeepBoop: you called {{.HelpCommand}}
##### Supported commands:
{{- range .SupportedCommands }}
- {{ . }};
{{- end }}

{{ with .StartHelp }}
@my_tag {{ .StartCommand }} HOST COMMIT_ID WORKFLOW_PATH [-e ENV=value]

OPTIONAL PARAMS:
- WORKFLOW_PATH
- `-e ENV=value`

#### Ex: `@my_tag {{ .StartCommand }} h200 SOME_SHA .github/workflows/dynamic-gpu-test.yml -e TEST_CASE=kandinsky5`

#### Ex: `@my_tag {{ .StartCommand }} h200 SOME_SHA .github/workflows/global-gpu-test.yml`
{{ end -}}

#### Supported hosts:
{{- range $key, $value := .Hosts.Hosts }}
#### ======Name=========
#### {{ $key }}
===================
{{- end }}
{{- end }}