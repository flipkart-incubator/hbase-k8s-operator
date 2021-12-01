{{- define "hbasecluster.dnscript" }}
#! /bin/bash
set -x -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

function shutdown() {
  while true; do
    #TODO: Kill it beyond certain wait time
    if [[ -f "/lifecycle/rs-terminated" ]]; then
      echo "Stopping datanode"
      sleep 3
      $HADOOP_HOME/bin/hdfs --daemon stop datanode
      break
    fi
    echo "Waiting for regionserver to die"
    sleep 2
  done
}

trap shutdown SIGTERM
exec $HADOOP_HOME/bin/hdfs datanode &
PID=$!

#TODO: Correct way to identify if process is up
touch /lifecycle/dn-started

wait $PID
{{- end }}
