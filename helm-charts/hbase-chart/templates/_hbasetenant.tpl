{{ define "com.flipkart.hbasetenant" }}
apiVersion: kvstore.flipkart.com/v1
kind: HbaseTenant
metadata:
  name: {{ .Values.service.name }}
  namespace: {{ .Values.namespace }}
spec:
  baseImage: {{ .Values.service.image }}
  fsgroup: {{ .Values.service.runAsGroup }}
  configuration:
    hbaseConfigName: {{ .Values.configuration.hbaseConfigName }}
    hbaseConfigMountPath: {{ .Values.configuration.hbaseConfigMountPath }}
    hbaseConfig:
      {{- $configPath := .Values.configuration.hbaseConfigRelPath }}
      {{- ($.Files.Glob $configPath).AsConfig | nindent 6 }}
    hadoopConfigName: {{ .Values.configuration.hadoopConfigName }}
    hadoopConfigMountPath: {{ .Values.configuration.hadoopConfigMountPath }}
    hadoopConfig:
      {{- $configPath := .Values.configuration.hadoopConfigRelPath }}
      {{- ($.Files.Glob $configPath).AsConfig | nindent 6 }}
  datanode: 
    {{- $dnsContainer := include "hbasecluster.dnslookup" . | indent 2 }}
    {{- $refreshNamenodeContainer := include "hbasecluster.refreshnn" . | indent 2 }}
    {{- $initContainers := list $dnsContainer $refreshNamenodeContainer }}

    {{- if .Values.commands.faultDomainCommand }}
    {{- $faultdomainContainer := include "hbasecluster.faultdomain" . | indent 2 }}
    {{- $initContainers = list $dnsContainer $faultdomainContainer $refreshNamenodeContainer }}
    {{- end }}

    {{- $dnscript := include "hbasecluster.dntenantscript" . | indent 6 }}
    {{- $rsscript := include "hbasecluster.rstenantscript" . | indent 6 }}
    {{- $dnprobescript := include "hbasecluster.dnprobescript" . | indent 8 }}
    {{- $ports1 := list 9866}}
    {{- $ports2 := list 16030 16020}}
    {{- $portsArr := list $ports1 $ports2}}
    {{- $scripts := list $dnscript $rsscript }}
    {{- $probescripts := list $dnprobescript "" }}
    {{- $arg1 := list .Values.configuration.hadoopLogPath .Values.configuration.hadoopConfigMountPath .Values.configuration.hadoopHomePath .Values.configuration.hadoopConfigName }}
    {{- $arg2 := list .Values.configuration.hbaseLogPath .Values.configuration.hbaseConfigMountPath .Values.configuration.hbaseHomePath .Values.configuration.hbaseConfigName }}
    {{- $args := list $arg1 $arg2 }}
    {{- $data := dict "Values" .Values "root" .Values.datanode "scripts" $scripts "probescripts" $probescripts "initContainers" $initContainers "args" $args "portsArr" $portsArr }}
    {{- include "hbasecluster.component" $data | indent 4 }}
{{- end }}
