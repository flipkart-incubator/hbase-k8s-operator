{{- define "hbasecluster.nnprobescript" }}
#! /bin/bash
set -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

if [[ $($HADOOP_HOME/bin/hdfs dfsadmin -safemode get | grep "Safe mode is OFF" | wc -l) == 0 ]]; then
  echo "Looks like there is no namenode with safemode off. Skipping checks..."
  exit 0
elif [[ $($HADOOP_HOME/bin/hdfs dfsadmin -safemode get | grep "$(hostname -f)" | grep "Safe mode is OFF" | wc -l) == 1 ]]; then
  echo "Namenode is out of safemode. Exiting..."
  exit 0
else
  echo "Namenode is still in safemode. Failing..."
  exit 1
fi

echo "Something unexpected happened at startup probe. Failing..."
exit 1
{{- end }}
