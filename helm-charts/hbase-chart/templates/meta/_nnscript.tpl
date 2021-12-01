{{- define "hbasecluster.nnscript" }}
#! /bin/bash
set -m -x

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

function shutdown() {
  echo "Stopping Namenode"
  is_active=$($HADOOP_HOME/bin/hdfs haadmin -getAllServiceState | grep "$(hostname -f)" | grep "active" | wc -l)

  if [[ $is_active == 1 ]]; then
    for i in $(echo $NNS | tr "," "\n"); do
      if [[ $($HADOOP_HOME/bin/hdfs haadmin -getServiceState $i | grep "standby" | wc -l) == 1 ]]; then
        STANDBY_SERVICE=$i
        break
      fi
    done

    echo "Is Active. Transitioning to standby"
    if [[ -n "$MY_SERVICE" && -n "$STANDBY_SERVICE" && $MY_SERVICE != $STANDBY_SERVICE ]]; then
      echo "Failing over from $MY_SERVICE to $STANDBY_SERVICE"
      $HADOOP_HOME/bin/hdfs haadmin -failover $MY_SERVICE $STANDBY_SERVICE
    else
      echo "$MY_SERVICE or $STANDBY_SERVICE is not defined or same. Cannot failover. Exitting..."
    fi
  else
   echo "Is not active"
  fi
  sleep 60
  echo "Completed shutdown cleanup"
  touch /lifecycle/nn-terminated
  $HADOOP_HOME/bin/hdfs --daemon stop namenode
}

NAMESERVICES=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.nameservices)
NNS=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.ha.namenodes.$NAMESERVICES)
MY_SERVICE=""
HTTP_ADDR=""
for i in $(echo $NNS | tr "," "\n"); do
  if [[ $($HADOOP_HOME/bin/hdfs getconf -confKey dfs.namenode.rpc-address.$NAMESERVICES.$i | sed 's/:[0-9]\+$//' | grep $(hostname -f) | wc -l ) == 1 ]]; then
    MY_SERVICE=$i
    HTTP_ADDR=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.namenode.http-address.$NAMESERVICES.$i)
  fi
done

echo "My Service: $MY_SERVICE"

trap shutdown SIGTERM
echo "N" | $HADOOP_HOME/bin/hdfs namenode -bootstrapStandby || true
exec $HADOOP_HOME/bin/hdfs namenode &
wait
{{- end }}
