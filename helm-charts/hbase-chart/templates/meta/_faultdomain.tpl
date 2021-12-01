{{- define "hbasecluster.faultdomain" }}
- name: init-faultdomain
  isBootstrap: false
  command:
  - /bin/bash
  - -c
  - |
    #! /bin/bash
    set -m -x

    export HBASE_LOG_DIR={{ .Values.configuration.hbaseLogPath }}
    export HBASE_CONF_DIR={{ .Values.configuration.hbaseConfigMountPath }}
    export HBASE_HOME={{ .Values.configuration.hbaseHomePath }}

    # Make it optional
    FAULT_DOMAIN_COMMAND={{ .Values.commands.faultDomainCommand | quote }}
    HOSTNAME=$(hostname -f)

    echo "Running command to get fault domain: $FAULT_DOMAIN_COMMAND"
    SMD=$(eval $FAULT_DOMAIN_COMMAND)
    echo "SMD value: $SMD"

    if [[ -n "$FAULT_DOMAIN_COMMAND" ]]; then
      echo "create /hbase-operator $SMD" | $HBASE_HOME/bin/hbase zkcli 2> /dev/null || true
      echo "create /hbase-operator/$HOSTNAME $SMD" | $HBASE_HOME/bin/hbase zkcli 2> /dev/null
      echo ""
      echo "Completed"
    fi
  cpuLimit: "0.1"
  memoryLimit: "386Mi"
  cpuRequest: "0.1"
  memoryRequest: "386Mi"
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
  volumeMounts:
  - name: nodeinfo
    mountPath: /etc/nodeinfo
    readOnly: true
{{- end }}
