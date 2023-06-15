{{- define "hbasecluster.dnscript" }}
#! /bin/bash
set -x -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

function shutdown() {
  while [[ ! -f "/lifecycle/rs-terminated" ]]; do echo "Waiting for regionserver to die"; sleep 2; done
  echo "Stopping datanode"
  sleep 10
  $HADOOP_HOME/bin/hdfs --daemon stop datanode
}

trap shutdown SIGTERM
exec $HADOOP_HOME/bin/hdfs datanode &
PID=$!

DOMAIN_SOCKET=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.domain.socket.path)
DOMAIN_SOCKET=$(echo $DOMAIN_SOCKET | sed -e 's/_PORT/*/g')
while [ ! -e ${DOMAIN_SOCKET} ]; do sleep 1; done
touch /lifecycle/dn-started

wait $PID
{{- end }}
