{{ define "hbasecluster.component" }}
  name: {{ .root.name }}
  size: {{ .root.replicas }}
  isPodServiceRequired: {{ default false .root.isPodServiceRequired }}
  shareProcessNamespace: {{ default false .root.shareProcessNamespace }}
  serviceAccountName: {{ default "default" .root.serviceAccountName }}
  {{- if .root.podManagementPolicy }}
  podManagementPolicy: {{ .root.podManagementPolicy }}
  {{- else if $.podManagementPolicy }}
  podManagementPolicy: {{ $.podManagementPolicy }}
  {{- else }}
  podManagementPolicy: "Parallel"
  {{- end }}
  {{- if .root.hostname }}
  hostname: {{ .root.hostname }}
  {{- end }}
  {{- if .root.subdomain }}
  subdomain: {{ .root.subdomain }}
  {{- end }}
  terminateGracePeriod: 120
  {{- if .root.labels }}
  labels:
  {{- range $key, $val := .root.labels }}
    {{ $key }}: {{ $val | quote }}
  {{- end }}
  {{- end }}
  {{- if .root.annotations }}
  annotations:
  {{- range $key, $val := .root.annotations }}
    {{ $key }}: {{ $val | quote }}
  {{- end }}
  {{- end }}
  {{- if .root.volumeClaims }}
  volumeClaims:
  {{- range .root.volumeClaims }}
  - name: {{ .name }}
    storageSize: {{ .size }}
    storageClassName: {{ .storageClass }}
    {{- if .annotations }}
    annotations:
    {{- range $key, $val := .annotations }}
      {{ $key }}: {{ $val | quote }}
    {{- end }}
    {{- end }}
    {{- if .labels }}
    labels:
    {{- range $key, $val := .labels }}
      {{ $key }}: {{ $val | quote }}
    {{- end }}
    {{- end }}
  {{- end }}
  {{- end }}
  {{- if .root.volumes }}
  volumes:
  {{- range .root.volumes }}
  - name: {{ .name }}
    volumeSource: {{ .volumeSource }}
    {{- if .path }}
    path: {{ .path }}
    {{- else if .secretName }}
    secretName: {{ .secretName }}
    {{- else if .configName }}
    configName: {{ .configName }}
    {{- end }}
  {{- end }}
  {{- end }}
  {{- if .root.dnsPolicy }}
  dnsPolicy: {{ .root.dnsPolicy }}
  {{- end }}
  {{- if .root.dnsConfig }}
  dnsConfig:
    nameservers:
    {{- if .root.dnsConfig.nameservers }}
    {{- range .root.dnsConfig.nameservers }}
    - {{ . }}
    {{- end }}
    {{- end }}
    searches:
    {{- if .root.dnsConfig.searches }}
    {{- range .root.dnsConfig.searches }}
    - {{ . }}
    {{- end }}
    {{- end }}
    options:
    {{- if .root.dnsConfig.options }}
    {{- range $index, $elem := .root.dnsConfig.options }}
    - name: {{ $elem.name }}
      value: {{ $elem.value | quote }}
    {{- end }}
    {{- end }}
  {{- end }}
  {{- if .root.hostAliases }}
  hostAliases:
  {{- range $index, $elem := .root.hostAliases }}
  - ip: {{ $elem.ip }}
    hostnames: {{ $elem.hostnames }}
  {{- end }}
  {{- end }}
  {{- if .root.topologySpreadConstraints }}
  topologySpreadConstraints:
  {{- range $index, $elem := .root.topologySpreadConstraints }}
  - maxSkew: {{ $elem.maxSkew }}
    topologyKey: {{ $elem.topologyKey }}
    whenUnsatisfiable: {{ default "DoNotSchedule" $elem.whenUnsatisfiable }}
  {{- end }}
  {{- end }}
  {{- if or .root.initContainers $.initContainers }}
  initContainers:
  {{- range $index, $elem := $.initContainers }}
  {{- . }}
  {{- end }}
  {{- range $index, $elem := .root.initContainers }}
  - name: {{ .name }}
    isBootstrap: {{ default false .isBootstrap }}
    command:
    - /bin/bash
    - -c
    - |
      {{- include $elem.templateName . | indent 6 }}
    cpuLimit: {{ .cpuLimit | quote }}
    memoryLimit: {{ .memoryLimit | quote }}
    cpuRequest: {{ .cpuRequest | quote }}
    memoryRequest: {{ .memoryRequest | quote }}
    securityContext:
      runAsUser: {{ $.Values.service.runAsUser }}
      runAsGroup: {{ $.Values.service.runAsGroup }}
    {{- if .volumeMounts }}
    volumeMounts:
    {{- range .volumeMounts }}
    - name: {{ .name }}
      mountPath: {{ .mountPath}}
      {{- if .readOnly }}
      readOnly: true
      {{- else }}
      readOnly: false
      {{- end }}
    {{- end }}
    {{- end }}
  {{- end }}
  {{- end }}
  {{- if .root.sidecarcontainers }}
  sidecarContainers:
  {{- range $index, $elem := .root.sidecarcontainers }}
  - name: {{ .name }}
    image: {{ .image }}
    {{- if .command }}
    command: {{ .command }}
    {{- end }}
    {{- if .args }}
    args:
    {{- range .args }}
    - {{ . | quote }}
    {{- end }}
    {{- end }}
    cpuLimit: {{ default "100m" .cpuLimit | quote }}
    memoryLimit: {{ default "128Mi" .memoryLimit | quote }}
    cpuRequest: {{ default "100m" .cpuRequest | quote }}
    memoryRequest: {{ default "128Mi" .memoryRequest | quote }}
    securityContext:
      runAsUser: {{ default $.Values.service.runAsUser .runAsUser}}
      runAsGroup: {{ default $.Values.service.runAsGroup .runAsGroup}}
    {{- if .volumeMounts }}
    volumeMounts:
    {{- range .volumeMounts }}
    - name: {{ .name }}
      mountPath: {{ .mountPath}}
      {{- if .readOnly }}
      readOnly: true
      {{- else }}
      readOnly: false
      {{- end }}
    {{- end }}
    {{- end }}
  {{- end }}
  {{- end }}
  {{- if .root.podDisruptionBudget }}
  podDisruptionBudget:
    {{- if .root.podDisruptionBudget.minAvailable }}
    minAvailable: {{ .root.podDisruptionBudget.minAvailable }}
    {{- else if .root.podDisruptionBudget.maxUnavailable }}
    maxUnavailable: {{ .root.podDisruptionBudget.maxUnavailable }}
    {{- end }}
  {{- end }}
  containers:
  {{- range $index, $elem := .root.containers }}
  {{- $parent := . }}
  {{- $ports := index $.portsArr $index }}
  {{- $probe := "" }}
  {{- if $.probescripts }}
  {{- $probe = index $.probescripts $index }}
  {{- end }}
  - name: {{ .name }}
    command:
    - /bin/bash
    - -c
    - |
      {{- index $.scripts $index }}
    args:
    {{- range index $.args $index }}
    - {{ . }}
    {{- end }}
    ports:
    {{- range $key, $val := $ports }}
    - port: {{ $val }}
      name: {{ $parent.name }}-{{ $key }}
    {{- end }}
    {{- if ne $probe "" }}
    startupProbe:
      initialDelay: {{ default 30 .startupProbeDelay }}
      timeout: 60
      failureThreshold: {{ default 10 .startupProbeFailureThreshold }}
      command:
      - /bin/bash
      - -c
      - |
      {{- $probe }}
      {{- range index $.args $index }}
      - {{ . }}
      {{- end }}
    {{- end }}
    livenessProbe:
      tcpPort: {{ first $ports }}
      initialDelay: {{ default 60 .probeDelay }}
    readinessProbe:
      tcpPort: {{ first $ports }}
      initialDelay: {{ default 60 .probeDelay }}
    cpuLimit: {{ .cpuLimit | quote }}
    memoryLimit: {{ .memoryLimit | quote }}
    cpuRequest: {{ .cpuRequest | quote }}
    memoryRequest: {{ .memoryRequest | quote }}
    securityContext:
      runAsUser: {{ $.Values.service.runAsUser }}
      runAsGroup: {{ $.Values.service.runAsGroup }}
      addSysPtrace: {{ default false $.root.shareProcessNamespace }}
    {{- if $parent.volumeMounts }}
    volumeMounts:
    {{- range $parent.volumeMounts }}
    - name: {{ .name }}
      mountPath: {{ .mountPath}}
      {{- if .readOnly }}
      readOnly: true
      {{- else }}
      readOnly: false
      {{- end }}
    {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
