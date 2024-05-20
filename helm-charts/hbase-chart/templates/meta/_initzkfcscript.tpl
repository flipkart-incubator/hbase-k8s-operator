{{- define "hbasecluster.initzkfcscript" }}
{{/* init containers which are required one time only during bootstrap  define their isBootstrap flag as true */}}
- name: init-zkfc
  isBootstrap: true
  {{- $zkfcCpu := "0.5" }}
  {{- $zkfcMemory := "512Mi" }}
  {{- range $key, $value := .Values.deployments.namenode.containers }}
    {{- if eq $value.name "zkfc" }}
      {{- $zkfcMemory = $value.memoryLimit }}
      {{- $zkfcCpu = $value.cpuLimit }}
    {{- end }}
  {{- end }}
  command:
  - /bin/bash
  - -c
  - |
    #! /bin/bash
    set -m -x

    export HADOOP_LOG_DIR={{ .Values.configuration.hadoopLogPath }}
    export HADOOP_CONF_DIR={{ .Values.configuration.hadoopConfigMountPath }}
    export HADOOP_HOME={{ .Values.configuration.hadoopHomePath }}

    echo "N" | $HADOOP_HOME/bin/hdfs zkfc -formatZK || true
  cpuLimit: {{ $zkfcCpu | quote }}
  memoryLimit: {{ $zkfcMemory | quote }}
  cpuRequest: {{ $zkfcCpu | quote }}
  memoryRequest: {{ $zkfcMemory | quote }}
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
{{- end }}
