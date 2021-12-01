{{- define "hbasecluster.initzkfcscript" }}
#! /bin/bash
set -m

export HADOOP_LOG_DIR=$0
export HADOOP_CONF_DIR=$1
export HADOOP_HOME=$2

echo "N" | $HADOOP_HOME/bin/hdfs zkfc -formatZK || true
{{- end }}
