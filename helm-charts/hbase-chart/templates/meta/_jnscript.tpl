{{- define "hbasecluster.jnscript" }}
#! /bin/bash
set -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2
export USER=$(whoami)
export HADOOP_LOG_FILE=$HADOOP_LOG_DIR/hadoop-$USER-journalnode-$(hostname).log

mkdir -p $HADOOP_LOG_DIR
touch $HADOOP_LOG_FILE

function shutdown() {
  echo "Stopping Journalnode"
  $HADOOP_HOME/bin/hdfs --daemon stop journalnode
}

trap shutdown SIGTERM
exec $HADOOP_HOME/bin/hdfs journalnode start 2>&1 | tee -a $HADOOP_LOG_FILE &
wait
{{- end }}
