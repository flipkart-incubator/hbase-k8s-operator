{{- define "hbasecluster.rstenantscript" }}
#! /bin/bash
set -m
export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2
export HBASE_CONF_NAME=$3
export USER=$(whoami)

mkdir -p $HBASE_LOG_DIR
ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out
ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log

function shutdown() {
  echo "Stopping Regionserver"
  host=`hostname -f`
  $HBASE_HOME/bin/hbase org.apache.hadoop.hbase.util.RegionMover -m 6 -r $host -o unload
  touch /lifecycle/rs-terminated
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

curl -sX GET http://127.0.0.1:8802/v1/configmaps/$HBASE_CONF_NAME | jq '.data | to_entries[] | .key, .value' | while IFS= read -r key; read -r value; do echo $value | jq -r '.' | tee $(echo $key | jq -r '.' | xargs -I {} echo $HBASE_CONF_DIR/{}) > /dev/null; done

sleep 1

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start regionserver &
wait
{{- end }}
