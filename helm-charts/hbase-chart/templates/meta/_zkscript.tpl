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
    IFS=$'\n' read -d '' -ra ZKs <<< $($HBASE_HOME/bin/hbase zkcli config 2> /dev/null  | grep participant | grep -o '=.*:2888' | cut -d : -f 1 | cut -c 2-)
    function leaderElection() {
       for zk in "${ZKs[@]}"; do
           myhost=$(echo $zk 2181)
           if [[ $zk != $(hostname -f) && $(echo "stat" | nc $myhost | grep "Mode: leader") ]]; then
             echo "$zk is leader"
             echo "Leader election completed"
             exit 0
           fi
       done
       }

    pod_timeout=120
    endTime=$(( $(date +%s) + $pod_timeout ))
    while [ $(date +%s) -lt $endTime ]; do
      leaderElection
      sleep 1
    done
    echo "Leader election did not complete but this zookeeper pod is shutting down as pod timeout is breached"
  fi
}

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start zookeeper &
wait
{{- end }}
