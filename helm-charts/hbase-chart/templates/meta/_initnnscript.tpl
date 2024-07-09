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

    while true; do
      echo "N" | $HADOOP_HOME/bin/hdfs namenode -format $($HADOOP_HOME/bin/hdfs getconf -confKey dfs.nameservices) ; exit_code=$?
      # If the format command was successful, break the loop
      if [ $exit_code -eq 0 ]; then
        echo "Command succeeded with exit status $exit_status, breaking the loop."
        break
      else
        # If the format command was not successful, check if there is any active namenode
        output=$($HADOOP_HOME/bin/hdfs haadmin -getAllServiceState)
        # If there is an active namenode, break the loop
        if echo "$output" | grep -q "active"; then
          echo "Active namenode found, breaking the loop."
          break
        else
          # If there is no active namenode, retry the format command
          echo "Command failed with exit status $exit_status and no active namenode found, retrying..."
          # Sleep for a random time between 0 and 5 seconds . This is done to avoid the racing condition between different namenodes
          sleep_time=$((RANDOM % 6))
          echo "Sleeping for $sleep_time seconds"
          sleep $sleep_time
        fi
      fi
    done

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
