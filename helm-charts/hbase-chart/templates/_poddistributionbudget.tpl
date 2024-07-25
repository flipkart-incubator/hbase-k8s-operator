{{- define "com.flipkart.poddistributionbudgets" }}
---
{{- include "createPDB" (dict "deployment" .Values.deployments.zookeeper "pdbConfig" .Values.deployments.zookeeper.pdbConfig "component" "zk" "namespace" .Values.namespace "instance" .Values.service.name) }}
---
{{- include "createPDB" (dict "deployment" .Values.deployments.journalnode "pdbConfig" .Values.deployments.journalnode.pdbConfig "component" "jn" "namespace" .Values.namespace "instance" .Values.service.name) }}
---
{{- include "createPDB" (dict "deployment" .Values.deployments.hmaster "pdbConfig" .Values.deployments.hmaster.pdbConfig "component" "hm" "namespace" .Values.namespace "instance" .Values.service.name) }}
---
{{- include "createPDB" (dict "deployment" .Values.deployments.datanode "pdbConfig" .Values.deployments.datanode.pdbConfig "component" "dn" "namespace" .Values.namespace "instance" .Values.service.name) }}
---
{{- include "createPDB" (dict "deployment" .Values.deployments.namenode "pdbConfig" .Values.deployments.namenode.pdbConfig "component" "nn" "namespace" .Values.namespace "instance" .Values.service.name) }}
---
{{- end -}}