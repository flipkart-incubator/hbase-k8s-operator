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

    # Get the journal node URIs
    jn_uris=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.namenode.shared.edits.dir)

    # Extract the journal node hostnames and ports
    jn_hostnames_and_ports=${jn_uris#qjournal://}  # Remove the 'qjournal://' prefix
    jn_hostnames_and_ports=${jn_hostnames_and_ports%/*}  # Remove the '/<nameserviceID>' suffix

    # Split the hostnames and ports into an array
    IFS=';' read -ra jn_array <<< "$jn_hostnames_and_ports"

    # Access each journal node hostname using the array and wait until service is ready
    for jn in "${jn_array[@]}"; do
      jn_hostname=${jn%%:*}
      until nslookup $jn_hostname; do
        echo "Waiting for $jn_hostname to be ready..."
        sleep 5
      done
    done

    # Sleep for random time to avoid lock issue on journalnode quorum
    sleep_time=$((RANDOM % 180))
    echo "Sleeping for $sleep_time seconds"
    sleep $sleep_time

    echo "Formatting the namenode"
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
