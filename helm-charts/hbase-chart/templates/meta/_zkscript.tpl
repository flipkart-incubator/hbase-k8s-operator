{{- define "hbasecluster.zkscript" }}
#! /bin/bash
set -m -x

export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2
export USER=$(whoami)

mkdir -p $HBASE_LOG_DIR
touch $HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).log &&  tail -F $HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).log &
touch $HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).out &&  tail -F $HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).out &

function shutdown() {
  echo "Stopping Zookeeper"
  $HBASE_HOME/bin/hbase-daemon.sh stop zookeeper
}

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start zookeeper &
wait
{{- end }}
