{{- define "hbasecluster.initnnbootstrapscript" }}
- name: init-nn-bootstrap-standby
  isBootstrap: false
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

    $HADOOP_HOME/bin/hdfs namenode -metadataVersion 2>&1; exit_code=$?
    if [ $exit_code -eq 1 ]
    then
      echo "Namenode metadata is not accessible , running bootstrap standby"
      $HADOOP_HOME/bin/hdfs namenode -bootstrapStandby -nonInteractive
    else
      echo "Namenode metadata is accessible , so skipping bootstrap"
    fi
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
