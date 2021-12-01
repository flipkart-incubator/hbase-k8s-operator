{{- define "hbasecluster.zkfcscript" }}
#! /bin/bash
set -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

function shutdown() {
  while true; do
    if [[ -f "/lifecycle/nn-terminated" ]]; then
      echo "Stopping zkfc"
      sleep 10
      $HADOOP_HOME/bin/hdfs --daemon stop zkfc
      break
    fi
    echo "Waiting for namenode to die"
    sleep 2
  done
}

trap shutdown SIGTERM
exec $HADOOP_HOME/bin/hdfs zkfc &
wait
{{- end }}
