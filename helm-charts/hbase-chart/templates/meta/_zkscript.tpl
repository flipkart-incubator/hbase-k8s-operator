{{- define "hbasecluster.zkscript" }}
#! /bin/bash
set -m -x
set -o pipefail

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
     if [ ! -f "$HBASE_CONF_DIR/hbase-site.xml" ]; then
        echo "$HBASE_CONF_DIR/hbase-site.xml does not exists"
        sleep 120
        exit 1
     fi

     if ! grep -q "<name>hbase.zookeeper.quorum</name>" "$HBASE_CONF_DIR/hbase-site.xml"; then
        echo "Error: $HBASE_CONF_DIR/hbase-site.xml does not contain <name>hbase.zookeeper.quorum</name>."
        sleep 120
        exit 1
     fi

     quorum=$(grep -A1 "<name>hbase.zookeeper.quorum</name>" "$HBASE_CONF_DIR/hbase-site.xml" |  grep "<value>" | sed -e 's/<value>\(.*\)<\/value>/\1/' | xargs)
     if [ -z "$quorum" ]; then
        echo "Error: <value> for <name>hbase.zookeeper.quorum</name> in $HBASE_CONF_DIR/hbase-site.xml is empty or not properly formatted."
        sleep 120
        exit 1
     fi

     if ! echo "$quorum" | grep -q ","; then
        echo "Error: <value> for <name>hbase.zookeeper.quorum</name> in $HBASE_CONF_DIR/hbase-site.xml does not appear to be a comma-separated list of zookeepers"
        sleep 120
        exit 1
     fi

     IFS=',' read -ra ZKs <<< $quorum
     if [[ ${#ZKs[@]} -gt 0 ]]; then
         function leaderElection() {
            for zk in "${ZKs[@]}"; do
               if [[ $zk != $(hostname -f) ]]; then
                  host=$(echo $zk 2181)
                  mode=$(echo "stat" | timeout 1s nc $host | grep "Mode: " | sed 's/Mode: //' | sed -e 's/[[:space:]]*$//')
                  if [[ $? -eq 0 && -n "$mode" ]]; then
                     if [[ $mode == leader ]]; then
                       echo "$zk is a $mode"
                       echo "Leader election completed"
                       exit 0
                     else
                       echo "$zk is a $mode"
                     fi
                  fi
               fi
            done
            }

         pod_timeout=110
         endTime=$(( $(date +%s) + $pod_timeout ))
         while [ $(date +%s) -lt $endTime ]; do
           leaderElection
           sleep 1
         done
         echo "Leader election did not complete but this zookeeper pod is shutting down as pod timeout is breached"
         exit 1
     else
       sleep 120
       exit 1
     fi
  fi
}

trap shutdown SIGTERM
exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start zookeeper &
wait
{{- end }}
