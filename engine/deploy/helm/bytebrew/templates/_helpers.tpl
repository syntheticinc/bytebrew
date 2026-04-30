{{/*
Expand the name of the chart.
*/}}
{{- define "bytebrew-engine.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "bytebrew-engine.fullname" -}}
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
{{- define "bytebrew-engine.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "bytebrew-engine.labels" -}}
helm.sh/chart: {{ include "bytebrew-engine.chart" . }}
{{ include "bytebrew-engine.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "bytebrew-engine.selectorLabels" -}}
app.kubernetes.io/name: {{ include "bytebrew-engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Build the DATABASE_URL from postgresql values.
Only used when postgresql.external.existingSecret is not set.
*/}}
{{- define "bytebrew-engine.databaseURL" -}}
{{- with .Values.postgresql.external -}}
postgres://{{ .username }}:{{ .password }}@{{ .host }}:{{ .port }}/{{ .database }}?sslmode={{ .sslmode }}
{{- end }}
{{- end }}

{{/*
Return the name of the Secret that holds DATABASE_URL.
Uses existingSecret when provided, otherwise the chart-managed Secret.
*/}}
{{- define "bytebrew-engine.databaseSecretName" -}}
{{- if .Values.postgresql.external.existingSecret -}}
{{- .Values.postgresql.external.existingSecret -}}
{{- else -}}
{{- printf "%s-secrets" (include "bytebrew-engine.fullname" .) -}}
{{- end -}}
{{- end }}

{{/*
Return the Secret key that holds DATABASE_URL.
*/}}
{{- define "bytebrew-engine.databaseSecretKey" -}}
{{- if .Values.postgresql.external.existingSecret -}}
{{- .Values.postgresql.external.existingSecretKey | default "DATABASE_URL" -}}
{{- else -}}
database-url
{{- end -}}
{{- end }}

{{/*
Return the ServiceAccount name.
*/}}
{{- define "bytebrew-engine.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "bytebrew-engine.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end }}
