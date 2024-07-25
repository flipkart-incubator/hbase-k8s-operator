{{- define "createPDB" }}
{{- $deployment := .deployment }}
{{- $pdbConfig := .pdbConfig }}
{{- $namespace := .namespace }}
{{- $instanceName := .instance }}
{{- if and $deployment (not $pdbConfig.pdbDisabled) }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  labels:
    app.kubernetes.io/component: {{ .component }}
    app.kubernetes.io/instance: pdb-{{ $deployment.name }}
  name: {{ $deployment.name }}-pdb
  namespace: {{ $namespace }}
spec:
  {{- if $pdbConfig.minAvailable }}
  minAvailable: {{ $pdbConfig.minAvailable | default 3 }}
  {{- else}}
  maxUnavailable: {{ $pdbConfig.maxUnavailable | default 1 }}
  {{- end }}
  selector:
    matchLabels:
      app.kubernetes.io/component: {{ .component }}
      app.kubernetes.io/instance: hc-{{ $instanceName }}
{{- end }}
{{- end }}