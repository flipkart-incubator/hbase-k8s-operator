{{- define "hbasecluster.initnnbootstrapscript" }}
- name: init-nn-bootstrap-standby
  isBootstrap: false
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
      echo "Namenode metadata is not accessible , checking for fsck health"
      $HADOOP_HOME/bin/hdfs fsck /hbase 2>&1; exit_code=$?
      if [ $exit_code -eq 0 ]
      then
        echo "Fsck is healthy, so bootstrapping standby namenode"
        $HADOOP_HOME/bin/hdfs namenode -bootstrapStandby -nonInteractive
      else
        echo "Namenode metadata is not accessible and fsck is not healthy, exiting !"
        exit 1
      fi
    else
      echo "Namenode metadata is accessible"
    fi
  cpuLimit: "0.5"
  memoryLimit: "512Mi"
  cpuRequest: "0.5"
  memoryRequest: "512Mi"
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
  volumeMounts:
  - name: {{ .Values.mount.namenodeMountName }}
    mountPath: {{ .Values.mount.namenodeMountPath }}
{{- end }}