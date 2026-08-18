{{- define "hbasecluster.hmasterscript" }}
#! /bin/bash
set -m
export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2
export USER=$(whoami)
{{- if .Values.configuration.hbaseHeapDumpPath }}
export HBASE_HEAPDUMP_PATH="{{ .Values.configuration.hbaseHeapDumpPath }}"
{{- end }}

mkdir -p $HBASE_LOG_DIR
touch $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log && tail -F $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log &
touch $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out && tail -F $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out &

function shutdown() {
  # stop alone does not clear /hbase/master; without clear the backup waits
  # ~zookeeper.session.timeout (~60s). Kubernetes signals PID 1 only, so both
  # steps run here before the container is torn down.
  echo "Stopping HMaster"
  $HBASE_HOME/bin/hbase-daemon.sh stop master

  # deleteIfEquals: removes /hbase/master only if it still names this server.
  PIDDIR=$(. "$HBASE_CONF_DIR/hbase-env.sh" >/dev/null 2>&1; echo "${HBASE_PID_DIR:-/tmp}")
  export HBASE_ZNODE_FILE="$PIDDIR/hbase-${USER}-master.znode"
  if [ -f "$HBASE_ZNODE_FILE" ]; then
    echo "Clearing master znode"
    ( cd "$HBASE_LOG_DIR" && timeout 30 $HBASE_HOME/bin/hbase master clear )
  fi
  echo "HMaster stop completed"
}

trap shutdown SIGTERM SIGINT
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start master &
wait
{{- end }}