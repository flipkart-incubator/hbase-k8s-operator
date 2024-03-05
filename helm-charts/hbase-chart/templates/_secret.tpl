{{- define "com.flipkart.secrets" }}
---
apiVersion: v1
kind: Secret
type: kubernetes.io/service-account-token
metadata:
  name: {{ .Values.secrets.configs.name }}
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/service-account.name: {{ .Values.secrets.configs.tokenServiceAccount }}
data:
  {{ .Values.secrets.configs.key }}: {{ .Values.secrets.configs.value }}
{{- end -}}