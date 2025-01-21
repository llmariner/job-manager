{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "job-manager-syncer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "job-manager-syncer.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "job-manager-syncer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "job-manager-syncer.labels" -}}
helm.sh/chart: {{ include "job-manager-syncer.chart" . }}
{{ include "job-manager-syncer.selectorLabels" . }}
{{- if .Values.version }}
app.kubernetes.io/version: {{ .Values.version | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "job-manager-syncer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "job-manager-syncer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "job-manager-syncer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "job-manager-syncer.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
metrics port
*/}}
{{- define "job-manager-syncer.metricsPort" -}}
{{ mustRegexSplit ":" .Values.kubernetesManager.metricsBindAddress -1 | last }}
{{- end -}}

{{/*
health port
*/}}
{{- define "job-manager-syncer.healthPort" -}}
{{ mustRegexSplit ":" .Values.kubernetesManager.healthBindAddress -1 | last }}
{{- end -}}

{{/*
Do nothing, just for validation.
*/}}
{{ .Values.enable }}
