{{- define "hbasecluster.dnprobescript" }}
#! /bin/bash
set -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

while :
do
  if [[ $($HADOOP_HOME/bin/hdfs dfsadmin -report -live | grep "$(hostname -f)" | wc -l) == 2 ]]; then
    echo "datanode is listed as live under namenode. Exiting..."
    exit 0
  else
    echo "datanode is still not listed as live under namenode"
    exit 1
  fi
done
exit 1
{{- end }}
