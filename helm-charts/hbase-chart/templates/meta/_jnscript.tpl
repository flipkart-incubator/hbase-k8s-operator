{{- define "hbasecluster.jnscript" }}
#! /bin/bash
set -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

function shutdown() {
  echo "Stopping Journalnode"
  $HADOOP_HOME/bin/hdfs --daemon stop journalnode
}

trap shutdown SIGTERM
exec $HADOOP_HOME/bin/hdfs journalnode start &
wait
{{- end }}
