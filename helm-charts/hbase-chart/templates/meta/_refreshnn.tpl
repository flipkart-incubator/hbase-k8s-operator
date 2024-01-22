{{- define "hbasecluster.refreshnn" }}
- name: init-refreshnn
  isBootstrap: false
  command:
  - /bin/bash
  - -c
  - |
    #! /bin/bash
    set -x -m

    export HADOOP_LOG_DIR={{ .Values.configuration.hadoopLogPath }}
    export HADOOP_CONF_DIR={{ .Values.configuration.hadoopConfigMountPath }}
    export HADOOP_HOME={{ .Values.configuration.hadoopHomePath }}

    $HADOOP_HOME/bin/hdfs dfsadmin -refreshNodes || true

  cpuLimit: "0.2"
  memoryLimit: "256Mi"
  cpuRequest: "0.2"
  memoryRequest: "256Mi"
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
{{- end }}
