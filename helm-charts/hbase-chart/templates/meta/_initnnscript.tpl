{{- define "hbasecluster.initnnscript" }}
{{/* init containers which are required one time only during bootstrap  define their isBootstrap flag as true */}}
- name: init-namenode
  isBootstrap: true
  {{- $namenodeCpu := "0.5" }}
  {{- $namenodeMemory := "512Mi" }}
  {{- range $key, $value := .Values.deployments.namenode.containers }}
    {{- if eq $value.name "namenode" }}
      {{- $namenodeMemory = $value.memoryLimit }}
      {{- $namenodeCpu = $value.cpuLimit }}
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

    SLEEP_TIME=$(( RANDOM % 120 ))
    echo "Sleeping for $SLEEP_TIME seconds to reduce race risk."
    sleep $SLEEP_TIME

    echo "N" | $HADOOP_HOME/bin/hdfs namenode -format $($HADOOP_HOME/bin/hdfs getconf -confKey dfs.nameservices) || true
  cpuLimit: {{ $namenodeCpu | quote }}
  memoryLimit: {{ $namenodeMemory | quote }}
  cpuRequest: {{ $namenodeCpu | quote }}
  memoryRequest: {{ $namenodeMemory | quote }}
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
  volumeMounts:
  - name: {{ .Values.mount.namenodeMountName }}
    mountPath: {{ .Values.mount.namenodeMountPath }}
{{- end }}
