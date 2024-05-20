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
  echo stat | nc localhost 2181 | grep "Mode: follower"
  exit_status=$?
  echo "Stopping Zookeeper"
  $HBASE_HOME/bin/hbase-daemon.sh stop zookeeper
  if [ $exit_status != 0 ]; then
	  echo "Leader stopped, sleeping for 120 seconds"
	  sleep 120
  fi
}

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start zookeeper &
wait
{{- end }}
