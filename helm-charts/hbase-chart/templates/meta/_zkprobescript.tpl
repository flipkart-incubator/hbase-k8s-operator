{{- define "hbasecluster.zkprobescript" }}
#! /bin/bash
set -m

export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2

#TODO: Find better alternative
IFS=',' read -ra ZKs <<< $($HBASE_HOME/bin/hbase zkcli quit 2> /dev/null | grep "Connecting to" | sed 's/Connecting to //')
visited=""
quorum=""
myhost="localhost 2181"
for zk in "${ZKs[@]}"; do
  if [[ $(echo $zk | grep $(hostname -f) | wc -l) == 1 ]]; then
    myhost=$(echo $zk | sed 's/:/ /')
  fi

  if [[ $(echo "stat" | nc $(echo $zk | sed 's/:/ /') | grep "Mode: " | wc -l) == 1 ]]; then
    quorum="present"
  fi
  visited="true"
done

if [[ -n $visited && -z $quorum ]]; then
  echo "Quorum is absent, disabling startup checks..."
  sleep 5
  exit 0
fi

if [[ $(echo "stat" | nc $myhost | grep "Mode: " | wc -l) == 1 ]]; then
  exit 0
else
  echo "zookeeper is not able to connect to quorum"
  exit 1
fi
{{- end }}
