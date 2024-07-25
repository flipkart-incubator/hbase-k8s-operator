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
    IFS=$'\n' read -d '' -ra ZKs <<< $($HBASE_HOME/bin/hbase zkcli config 2> /dev/null  | grep ^server. | grep -o '=.*:' | cut -d : -f 1 | cut -c 2-)
    function leaderElection() {
       for zk in "${ZKs[@]}"; do
           host=$(echo $zk 2181)
           zk_mode=$(echo "stat" | nc $host | grep "Mode: leader")
           if [[ $zk != $(hostname -f) && $zk_mode ]]; then
             echo "$zk is a leader"
             echo "Leader election completed"
             exit 0
           else
             echo "$zk is a follower"
           fi
       done
       }

    pod_timeout=20
    endTime=$(( $(date +%s) + $pod_timeout ))
    while [ $(date +%s) -lt $endTime ]; do
      leaderElection
      sleep 1
    done
    echo "Leader election did not complete but this zookeeper pod is shutting down as pod timeout is breached"
    exit 1
  fi
}

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start zookeeper &
wait
{{- end }}
