{{- define "hbasecluster.dnslookup" }}
- name: init-dnslookup
  isBootstrap: false
  command:
  - /bin/bash
  - -c
  - |
    #! /bin/bash
    set -m

    i=0
    while true; do
      echo "$i iteration"
      dig +short $(hostname -f) | grep -v -e '^$'
      if [ $? == 0 ]; then
        sleep 30 # 30 seconds default dns caching
        echo "Breaking..."
        break
      fi
      i=$((i + 1))
      sleep 1
    done
  cpuLimit: "0.2"
  memoryLimit: "128Mi"
  cpuRequest: "0.2"
  memoryRequest: "128Mi"
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
{{- end }}
