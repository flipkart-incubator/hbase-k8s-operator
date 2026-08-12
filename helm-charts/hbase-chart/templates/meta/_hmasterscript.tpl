{{- define "hbasecluster.hmasterscript" }}
#! /bin/bash
set -m
export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2
export USER=$(whoami)

mkdir -p $HBASE_LOG_DIR
touch $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log && tail -F $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log &
touch $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out && tail -F $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out &

function shutdown() {
  # Graceful stop: SIGTERM the master JVM and wait until it exits so
  # ActiveMasterManager can delete /hbase/master (or close the ZK session)
  # before this container dies. Without waiting, kubelet SIGKILLs the JVM
  # and backup waits ~zookeeper.session.timeout (~60s) for failover.
  echo "Stopping Hmaster"
  MPID=$(ps -eo pid,args | awk '/[D]proc_master/ {print $1; exit}')
  if [ -n "$MPID" ]; then
    kill -TERM "$MPID"
    echo "Sent SIGTERM to master JVM pid=$MPID; waiting for exit"
    while kill -0 "$MPID" 2>/dev/null; do
      sleep 1
    done
    echo "Hmaster JVM exited"
  else
    echo "Master JVM not found; falling back to hbase-daemon.sh stop master"
    $HBASE_HOME/bin/hbase-daemon.sh stop master
  fi
}

trap shutdown SIGTERM SIGINT
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start master &
wait
{{- end }}
