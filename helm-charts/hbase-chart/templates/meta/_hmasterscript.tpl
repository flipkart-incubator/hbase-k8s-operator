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
  # HMaster does not remove its /hbase/master znode on SIGTERM, so a backup
  # cannot take over until zookeeper.session.timeout expires (~60s).
  # hbase-daemon.sh clears the znode from its EXIT trap, which runs too late
  # during container teardown. Kubernetes signals PID 1 only, so the master
  # stop sequence is driven from here.
  echo "Stopping HMaster (graceful)"

  # HBASE_PID_DIR is defined in hbase-env.sh, which this script does not source.
  PIDDIR=$(. "$HBASE_CONF_DIR/hbase-env.sh" >/dev/null 2>&1; echo "${HBASE_PID_DIR:-/tmp}")
  PIDFILE="$PIDDIR/hbase-${USER}-master.pid"

  MPID=""
  if [ -f "$PIDFILE" ]; then
    MPID=$(cat "$PIDFILE")
  fi
  if [ -z "$MPID" ] || ! kill -0 "$MPID" 2>/dev/null; then
    MPID=$(ps -eo pid,args | awk '/[D]proc_master/ {print $1; exit}')
  fi

  if [ -n "$MPID" ] && kill -0 "$MPID" 2>/dev/null; then
    echo "SIGTERM master JVM pid=$MPID; waiting for exit"
    kill -TERM "$MPID" 2>/dev/null || true
    # The wait is bounded because the znode must only be cleared after the JVM
    # has exited, and an unbounded wait would consume the termination grace
    # period without ever clearing it.
    for _ in $(seq 1 "${HBASE_MASTER_STOP_TIMEOUT:-30}"); do
      kill -0 "$MPID" 2>/dev/null || break
      sleep 1
    done
    if kill -0 "$MPID" 2>/dev/null; then
      echo "JVM still up after timeout; sending SIGKILL"
      kill -9 "$MPID" 2>/dev/null || true
      while kill -0 "$MPID" 2>/dev/null; do sleep 1; done
    fi
    echo "Master JVM exited"
  else
    echo "Master JVM not running"
  fi

  # Clearing the znode hands over immediately instead of waiting for the ZK
  # session to expire. ZNodeClearer requires HBASE_ZNODE_FILE and silently does
  # nothing without it. Its deleteIfEquals only removes a znode that still
  # points at this server, so another active master can never be evicted. Only
  # an active master writes the file, so backup pods skip this step.
  export HBASE_ZNODE_FILE="$PIDDIR/hbase-${USER}-master.znode"
  if [ -f "$HBASE_ZNODE_FILE" ]; then
    echo "Clearing master znode"
    # SERVER_GC_OPTS sets a -Xloggc path relative to cwd, and PID 1's cwd is
    # not writable, so the helper JVM must start from the log directory.
    ( cd "$HBASE_LOG_DIR" && timeout 30 $HBASE_HOME/bin/hbase master clear )
    echo "Master znode cleared rc=$?"
  fi
  echo "HMaster stop completed"
}

trap shutdown SIGTERM SIGINT
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start master &
wait
{{- end }}