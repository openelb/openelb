{{/*
Expand the name of the chart.
*/}}
{{- define "openelb.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Expand the chart plus release name (used by the chart label)
*/}}
{{- define "openelb.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version -}}
{{- end -}}

{{- define "openelb.namespace" -}}
{{- .Release.Namespace -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "openelb.fullname" -}}
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
Create a default fully qualified manager name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "openelb.controller.fullname" -}}
{{- printf "%s-%s" (include "openelb.fullname" .) "controller" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "openelb.speaker.fullname" -}}
{{- printf "%s-%s" (include "openelb.fullname" .) "speaker" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "openelb.admission.fullname" -}}
{{- printf "%s-%s" (include "openelb.fullname" .) "admission" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "openelb.keepalived.fullname" -}}
{{- printf "%s-%s" (include "openelb.fullname" .) "keepalived" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "openelb.labels" -}}
app.kubernetes.io/name: {{ include "openelb.name" . }}
helm.sh/chart: {{ include "openelb.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}


{{/*
Selector labels
*/}}
{{- define "openelb.controller.labels" -}}
{{- include "openelb.labels" . }}
app.kubernetes.io/component: {{ include "openelb.controller.fullname" . }}
{{- end -}}

{{- define "openelb.speaker.labels" -}}
{{- include "openelb.labels" . }}
app.kubernetes.io/component: {{ include "openelb.speaker.fullname" . }}
{{- end -}}

{{- define "openelb.admission.labels" -}}
{{- include "openelb.labels" . }}
app.kubernetes.io/component: {{ include "openelb.admission.fullname" . }}
{{- end -}}

{{- define "openelb.keepalived.labels" -}}
{{- include "openelb.labels" . }}
app.kubernetes.io/component: {{ include "openelb.keepalived.fullname" . }}
{{- end -}}

{{- define "openelb.controller.serviceAccountName" -}}
    {{ default "openelb-controller" .Values.controller.serviceAccountName }}
{{- end -}}

{{- define "openelb.speaker.serviceAccountName" -}}
    {{ default "openelb-speaker" .Values.speaker.serviceAccountName }}
{{- end -}}

{{- define "openelb.admission.serviceAccountName" -}}
    {{ default "openelb-admission" .Values.admission.serviceAccountName }}
{{- end -}}
