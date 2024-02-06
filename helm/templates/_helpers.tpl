
{{/*
Common labels
*/}}
{{- define "gloo-portal-idp-connect.labels" -}}
app: {{ .Values.fullname }}
{{- end }}

{{/*
gloo-portal-idp-connect args command
*/}}
{{- define "gloo-portal-idp-connect.cmd.args" -}}
{{- if eq .Values.connector "cognito"}}
  - cognito
  - --user-pool={{ .Values.cognito.userPoolId }}
  - --resource-server={{ .Values.cognito.resourceServer }}
{{- end }}
{{- end }}
