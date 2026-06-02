{{- define "hbasecluster.rsprobescript" }}
#! /bin/bash
set -m

export HBASE_LOG_DIR=$0
export HBASE_CONF_DIR=$1
export HBASE_HOME=$2

# Probe succeeds once the regionserver has bound its RPC port (16020),
# which signals it has finished starting up and is accepting connections.
if timeout 2 bash -c "exec 3<>/dev/tcp/localhost/16020" 2>/dev/null; then
  echo "regionserver is bound to RPC port. Exiting..."
  exit 0
else
  echo "regionserver is not yet bound to RPC port. Failing..."
  exit 1
fi
{{- end }}
