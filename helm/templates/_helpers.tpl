
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
  - --port=8080
  - --user-pool-id={{ .Values.cognito.userPoolId }}
  - --resource-server={{ .Values.cognito.resourceServer }}
{{- else if eq .Values.connector "keycloak"}}
  - keycloak
  - --port=8080
  - --issuer={{ .Values.keycloak.realm }}
  - --client-id={{ .Values.keycloak.mgmtClientId }}
  - --client-secret={{ .Values.keycloak.mgmtClientSecret }}
  - --resource-server={{ .Values.keycloak.resourceServer }}
{{- end }}
{{- end }}
