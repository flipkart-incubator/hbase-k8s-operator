{{- define "hbasecluster.rsscript" }}
#! /bin/bash
set -m
export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2
export USER=$(whoami)

FAULT_DOMAIN_COMMAND=$3

mkdir -p $HBASE_LOG_DIR
touch $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log && tail -F $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log &
touch $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out && tail -F $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out &

function shutdown() {
  echo "Stopping Regionserver"
  host=`hostname -f`
  export HBASE_STOP_TIMEOUT=20
  echo "swtich off balancer"
  echo "balance_switch false" | $HBASE_HOME/bin/hbase shell &>/tmp/null
  $HBASE_HOME/bin/hbase {{ default "org.apache.hadoop.hbase.util.RegionMover" .Values.configuration.regionMoverClass }} -m 6 -r $host -o unload
  sleep 5
  echo "swtich on balancer"
  echo "balance_switch true" | $HBASE_HOME/bin/hbase shell &>/tmp/null
  touch /lifecycle/rs-terminated
  echo "stopping server now"
  $HBASE_HOME/bin/hbase-daemon.sh stop regionserver
}

while true; do
  if [[ -f "/lifecycle/dn-started" ]]; then
    echo "Starting rs"
    sleep 5
    break
  fi
  echo "Waiting for datanode to start"
  sleep 2
done

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start regionserver &
wait
{{- end }}
