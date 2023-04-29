{{ define "com.flipkart.hbasecluster" }}
{{- if eq .Values.sharedWithOperatorNamespace true }}
---
{{- else }}
{{- include "com.flipkart.hbaseoperator.roles" . }}
---
{{- include "com.flipkart.hbaseoperator.rolebindings" . }}
---
{{- end }}
apiVersion: kvstore.flipkart.com/v1
kind: HbaseCluster
metadata:
  name: {{ .Values.service.name }}
  namespace: {{ .Values.namespace }}
spec:
  baseImage: {{ .Values.service.image }}
  isBootstrap: {{ default false .Values.service.isBootstrap }}
  fsgroup: {{ .Values.service.runAsGroup }}
  configuration:
    hbaseConfigName: {{ .Values.configuration.hbaseConfigName }}
    hbaseConfigMountPath: {{ .Values.configuration.hbaseConfigMountPath }}
    hbaseConfig:
      {{- $configPath := .Values.configuration.hbaseConfigRelPath }}
      {{- ($.Files.Glob $configPath).AsConfig | nindent 6 }}
    hbaseTenantConfig:
      {{- $tenantConfigPath := dir .Values.configuration.hbaseConfigRelPath }}
      {{- $tenantConfigPath := printf "%s/tenants/*/*" $tenantConfigPath }}
      {{- range $path, $_ := .Files.Glob  $tenantConfigPath }}
      {{- $dir := dir $path }}
    - namespace: {{ base $dir }}
      {{- ($.Files.Glob $path).AsConfig | nindent 6 }}
      {{ end }}
    hadoopConfigName: {{ .Values.configuration.hadoopConfigName }}
    hadoopConfigMountPath: {{ .Values.configuration.hadoopConfigMountPath }}
    hadoopConfig:
      {{- $configPath := .Values.configuration.hadoopConfigRelPath }}
      {{- ($.Files.Glob $configPath).AsConfig | nindent 6 }}
    hadoopTenantConfig:
      {{- $tenantConfigPath := dir .Values.configuration.hadoopConfigRelPath }}
      {{- $tenantConfigPath := printf "%s/tenants/*/*" $tenantConfigPath }}
      {{- range $path, $_ := .Files.Glob  $tenantConfigPath }}
      {{- $dir := dir $path }}
    - namespace: {{ base $dir }}
      {{- ($.Files.Glob $path).AsConfig | nindent 6 }}
      {{ end }}
  {{- if .Values.tenantNamespaces }}
  tenantNamespaces:
  {{- range .Values.tenantNamespaces }}
    - {{ . }}
  {{- end }}
  {{- end }}
  deployments:
    {{- $refreshNamenodeContainer := include "hbasecluster.refreshnn" . | indent 2 }}
    {{- $dnsContainer := include "hbasecluster.dnslookup" . | indent 2 }}
    {{- $initnnContainer := include "hbasecluster.initnnscript" . | indent 2 }}
    {{- $initzkfcContainer := include "hbasecluster.initzkfcscript" . | indent 2 }}
    {{- if .Values.deployments.zookeeper }}
    zookeeper: 
      {{- $podManagementPolicy := "Parallel" }}
      {{- $initContainers := list $dnsContainer }}
      {{- $zkscript := include "hbasecluster.zkscript" . | indent 6 }}
      {{- $zkprobescript := include "hbasecluster.zkprobescript" . | indent 8 }}
      {{- $ports := list 2181 2888 3888 }}
      {{- $portsArr := list $ports }}
      {{- $scripts := list $zkscript }}
      {{- $arg1 := list .Values.configuration.hbaseLogPath .Values.configuration.hbaseConfigMountPath .Values.configuration.hbaseHomePath }}
      {{- $args := list $arg1 }}
      {{- $probescripts := list $zkprobescript }}
      {{- $data := dict "Values" .Values "root" .Values.deployments.zookeeper "scripts" $scripts "initContainers" $initContainers "probescripts" $probescripts "args" $args "portsArr" $portsArr "podManagementPolicy" $podManagementPolicy }}
      {{- include "hbasecluster.component" $data | indent 4 }}
    {{- end }}
    journalnode: 
      {{- $podManagementPolicy := "Parallel" }}
      {{- $initContainers := list $dnsContainer }}
      {{- $jnscript := include "hbasecluster.jnscript" . | indent 6 }}
      {{- $ports := list 8485 8480 }}
      {{- $portsArr := list $ports }}
      {{- $scripts := list $jnscript }}
      {{- $arg1 := list .Values.configuration.hadoopLogPath .Values.configuration.hadoopConfigMountPath .Values.configuration.hadoopHomePath }}
      {{- $args := list $arg1 }}
      {{- $data := dict "Values" .Values "root" .Values.deployments.journalnode "scripts" $scripts "initContainers" $initContainers "args" $args "portsArr" $portsArr "podManagementPolicy" $podManagementPolicy }}
      {{- include "hbasecluster.component" $data | indent 4 }}
    hmaster: 
      {{- $podManagementPolicy := "Parallel" }}
      {{- $initContainers := list $dnsContainer }}
      {{- $hmasterscript := include "hbasecluster.hmasterscript" . | indent 6 }}
      {{- $ports := list 16000 16010}}
      {{- $portsArr := list $ports}}
      {{- $scripts := list $hmasterscript }}
      {{- $arg1 := list .Values.configuration.hbaseLogPath .Values.configuration.hbaseConfigMountPath .Values.configuration.hbaseHomePath }}
      {{- $args := list $arg1 }}
      {{- $data := dict "Values" .Values "root" .Values.deployments.hmaster "scripts" $scripts "initContainers" $initContainers "args" $args "portsArr" $portsArr "podManagementPolicy" $podManagementPolicy }}
      {{- include "hbasecluster.component" $data | indent 4 }}
    datanode: 
      {{- $podManagementPolicy := "Parallel" }}
      {{- $initContainers := list $dnsContainer $refreshNamenodeContainer }}
      {{- if .Values.commands.faultDomainCommand }}
      {{- $faultdomainContainer := include "hbasecluster.faultdomain" . | indent 2 }}
      {{- $initContainers = list $dnsContainer $faultdomainContainer $refreshNamenodeContainer }}
      {{- end }}
      {{- $dnscript := include "hbasecluster.dnscript" . | indent 6 }}
      {{- $rsscript := include "hbasecluster.rsscript" . | indent 6 }}
      {{- $dnprobescript := include "hbasecluster.dnprobescript" . | indent 8 }}
      {{- $ports1 := list 9866}}
      {{- $ports2 := list 16020 16030}}
      {{- $portsArr := list $ports1 $ports2}}
      {{- $scripts := list $dnscript $rsscript }}
      {{- $probescripts := list $dnprobescript "" }}
      {{- $arg1 := list .Values.configuration.hadoopLogPath .Values.configuration.hadoopConfigMountPath .Values.configuration.hadoopHomePath }}
      {{- $arg2 := list .Values.configuration.hbaseLogPath .Values.configuration.hbaseConfigMountPath .Values.configuration.hbaseHomePath }}
      {{- $args := list $arg1 $arg2 }}
      {{- $data := dict "Values" .Values "root" .Values.deployments.datanode "scripts" $scripts "probescripts" $probescripts "initContainers" $initContainers "args" $args "portsArr" $portsArr "podManagementPolicy" $podManagementPolicy }}
      {{- include "hbasecluster.component" $data | indent 4 }}
    namenode:
      {{- $podManagementPolicy := "OrderedReady" }}
      {{- $initContainers := list $dnsContainer $initnnContainer $initzkfcContainer }}
      {{- $nnscript := include "hbasecluster.nnscript" . | indent 6 }}
      {{- $zkfcscript := include "hbasecluster.zkfcscript" . | indent 6 }}
      {{- $nnprobescript := include "hbasecluster.nnprobescript" . | indent 8 }}
      {{- $ports1 := list 8020 9870 50070 9000 }}
      {{- $ports2 := list 8019 }}
      {{- $portsArr := list $ports1 $ports2}}
      {{- $scripts := list $nnscript $zkfcscript }}
      {{- $arg1 := list .Values.configuration.hadoopLogPath .Values.configuration.hadoopConfigMountPath .Values.configuration.hadoopHomePath }}
      {{- $arg2 := list .Values.configuration.hadoopLogPath .Values.configuration.hadoopConfigMountPath .Values.configuration.hadoopHomePath }}
      {{- $args := list $arg1 $arg2 }}
      {{- $probescripts := list $nnprobescript "" }}
      {{- $data := dict "Values" .Values "root" .Values.deployments.namenode "scripts" $scripts "initContainers" $initContainers "args" $args "probescripts" $probescripts "portsArr" $portsArr "podManagementPolicy" $podManagementPolicy }}
      {{- include "hbasecluster.component" $data | indent 4 }}
{{- end }}
