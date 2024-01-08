{{- define "hbasecluster.dntenantscript" }}
#! /bin/bash
set -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2
export HADOOP_CONF_NAME=$3
export USER=$(whoami)
export HADOOP_LOG_FILE=$HADOOP_LOG_DIR/hadoop-$USER-datanode-$(hostname).log

mkdir -p $HADOOP_LOG_DIR
touch $HADOOP_LOG_FILE

function shutdown() {
  while true; do
    #TODO: Kill it beyond certain wait time
    if [[ -f "/lifecycle/rs-terminated" ]]; then
      echo "Stopping datanode"
      sleep 3
      $HADOOP_HOME/bin/hdfs --daemon stop datanode
      break
    fi
    echo "Waiting for regionserver to die"
    sleep 2
  done
}

#move this to init container
curl -sX GET http://127.0.0.1:8802/v1/configmaps/$HADOOP_CONF_NAME | jq '.data | to_entries[] | .key, .value' | while IFS= read -r key; read -r value; do echo $value | jq -r '.' | tee $(echo $key | jq -r '.' | xargs -I {} echo $HADOOP_CONF_DIR/{}) > /dev/null; done

sleep 1

trap shutdown SIGTERM
exec $HADOOP_HOME/bin/hdfs datanode 2>&1 | tee -a $HADOOP_LOG_FILE &
PID=$!

#TODO: Correct way to identify if process is up
touch /lifecycle/dn-started

wait $PID
{{- end }}
