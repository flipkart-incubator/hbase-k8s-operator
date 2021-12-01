{{- define "com.flipkart.configmaps" }}

{{- range .Values.configMaps.configs }}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .name }}
data:
{{- if eq .source "dir"}}
{{- $configPath := printf "%s/%s/%s/*" "config" $.Values.configMaps.envName .sourceLoc }}
{{- ($.Files.Glob $configPath).AsConfig | nindent 2 }}
{{- end }}
{{- if eq .source "data"}}
{{- toYaml .data | nindent 2}}
{{- end }}
{{- if eq .source "file"}}
{{- $configFilePath := printf "%s/%s/%s" "config" $.Values.configMaps.envName .sourceLoc }}
{{- $.Files.Get $configFilePath | nindent 2 }}
{{- end }}
{{- end }}

{{- end -}}
