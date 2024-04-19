{{- define "hbasecluster.initnnbootstrapscript" }}
- name: init-namenode-bootstrap
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

    metadataVersionOutput=$($HADOOP_HOME/bin/hdfs namenode -metadataVersion 2>&1)
    if echo "$metadataVersionOutput" | grep -q "java.io.FileNotFoundException: /grid/1/dfs/nn/current/VERSION"
    then
      echo "Namenode Directory does not exist ,checking for fsck health"
      fsckOutput=$($HADOOP_HOME/bin/hdfs fsck /hbase 2>&1)
      if echo "$fsckOutput" | grep -q "Status: HEALTHY"
      then
        echo "Namenode Directory does not exist but fsck is healthy, so copying fsimage and edits from active namenode"
        echo "Y" | $HADOOP_HOME/bin/hdfs namenode -bootstrapStandby
      else
        echo "Namenode Directory does not exist and fsck is not healthy, so exiting"
        exit 1
      fi
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
