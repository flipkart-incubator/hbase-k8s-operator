{{- define "hbasecluster.dnregister" }}
- name: init-dnregister
  isBootstrap: false
  command:
  - /bin/bash
  - -c
  - |
    #! /bin/bash
    set -x -m

    export HBASE_LOG_DIR={{ .Values.configuration.hbaseLogPath }}
    export HBASE_CONF_DIR={{ .Values.configuration.hbaseConfigMountPath }}
    export HBASE_HOME={{ .Values.configuration.hbaseHomePath }}
    export HADOOP_LOG_DIR={{ .Values.configuration.hadoopLogPath }}
    export HADOOP_CONF_DIR={{ .Values.configuration.hadoopConfigMountPath }}
    export HADOOP_HOME={{ .Values.configuration.hadoopHomePath }}

    FAULT_DOMAIN_COMMAND={{ .Values.commands.faultDomainCommand | quote }}
    HOSTNAME=$(hostname -f)

    # Register fault domain in ZK so HBase can place regions rack-aware.
    echo "Running command to get fault domain: $FAULT_DOMAIN_COMMAND"
    SMD=$(eval $FAULT_DOMAIN_COMMAND)
    echo "SMD value: $SMD"

    if [[ -n "$FAULT_DOMAIN_COMMAND" ]]; then
      echo "create /hbase-operator $SMD" | $HBASE_HOME/bin/hbase zkcli 2> /dev/null || true
      echo "create /hbase-operator/$HOSTNAME $SMD" | $HBASE_HOME/bin/hbase zkcli 2>/dev/null || \
       echo "set /hbase-operator/$HOSTNAME $SMD" | $HBASE_HOME/bin/hbase zkcli 2>/dev/null  || {
        echo "ERROR: Failed to register fault domain in ZooKeeper for $HOSTNAME"
        exit 1
      }
      echo ""
      echo "Completed"
    fi

    # Refresh the  NN include-list so this DN is allowed to register.
    # 5s gap between the two calls lets the first refresh settle before the second.
    echo "Refreshing namenode include-list"
    $HADOOP_HOME/bin/hdfs dfsadmin -refreshNodes || true
    echo "Sleeping 5s before next refresh"
    sleep 5
    echo "Refreshing namenode include-list again"
    $HADOOP_HOME/bin/hdfs dfsadmin -refreshNodes || true
  cpuLimit: "2"
  memoryLimit: "1Gi"
  cpuRequest: "2"
  memoryRequest: "1Gi"
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
  volumeMounts:
  - name: nodeinfo
    mountPath: /etc/nodeinfo
    readOnly: true
{{- end }}
