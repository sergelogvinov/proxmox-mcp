{{/*
Expand the name of the chart.
*/}}
{{- define "proxmox-mcp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "proxmox-mcp.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "proxmox-mcp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "proxmox-mcp.labels" -}}
helm.sh/chart: {{ include "proxmox-mcp.chart" . }}
{{ include "proxmox-mcp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "proxmox-mcp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "proxmox-mcp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "proxmox-mcp.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "proxmox-mcp.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return a podAffinity/podAntiAffinity definition
*/}}
{{- define "affinities.pods" -}}
  {{- if eq .Values.podAntiAffinityPreset "soft" }}
    {{- include "affinities.pods.soft" . -}}
  {{- else if eq .Values.podAntiAffinityPreset "hard" }}
    {{- include "affinities.pods.hard" . -}}
  {{- end -}}
{{- end -}}

{{/*
Return a soft podAntiAffinity definition
*/}}
{{- define "affinities.pods.soft" -}}
podAntiAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
    - podAffinityTerm:
        labelSelector:
          matchLabels: {{- (include "proxmox-mcp.selectorLabels" .) | nindent 12 }}
        namespaces:
          - {{ .Release.Namespace | quote }}
        topologyKey: kubernetes.io/hostname
      weight: 1
{{- end -}}

{{/*
Return a hard podAntiAffinity definition
*/}}
{{- define "affinities.pods.hard" -}}
podAntiAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchLabels: {{- (include "proxmox-mcp.selectorLabels" .) | nindent 10 }}
      namespaces:
        - {{ .Release.Namespace | quote }}
      topologyKey: kubernetes.io/hostname
{{- end -}}
