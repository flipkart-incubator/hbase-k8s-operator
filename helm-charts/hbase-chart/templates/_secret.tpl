{{- define "com.flipkart.secrets" }}

{{- if .Values.secrets }}

{{- range .Values.secrets.configs }}
---
apiVersion: v1
kind: Secret
type: kubernetes.io/service-account-token
metadata:
  namespace: {{ $.Values.namespace }}
  annotations:
      kubernetes.io/service-account.name: {{ $.Values.serviceAccountName }}
  name: {{ .name }}
data:
   {{- range $key, $value := .token }}
   {{ $key }}: {{ $value }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}
