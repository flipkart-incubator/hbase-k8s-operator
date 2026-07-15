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
        sleep 15
        echo "Breaking..."
        break
      fi
      i=$((i + 1))
      sleep 1
    done
  cpuLimit: "1"
  memoryLimit: "512Mi"
  cpuRequest: "1"
  memoryRequest: "512Mi"
  securityContext:
    runAsUser: {{ .Values.service.runAsUser }}
    runAsGroup: {{ .Values.service.runAsGroup }}
{{- end }}
