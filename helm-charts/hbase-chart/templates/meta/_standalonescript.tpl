{{- define "hbasecluster.standalonescript" }}
#! /bin/bash
set -m
export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2
export USER=$(whoami)

mkdir -p $HBASE_LOG_DIR
touch $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log
tail -F $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log &

function shutdown() {
  echo "Stopping Standalone"
  $HBASE_HOME/bin/hbase-daemon.sh stop master
}

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start master &
wait
{{- end }}
