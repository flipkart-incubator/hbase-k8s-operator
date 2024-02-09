{{- define "hbasecluster.initzkfcscript" }}
{{/* init containers which are required one time only during bootstrap  define their isBootstrap flag as true */}}
- name: init-zkfc
  isBootstrap: true
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
  cpuLimit: "0.5"
  memoryLimit: "512Mi"
  cpuRequest: "0.5"
  memoryRequest: "512Mi"
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
{{- end }}
