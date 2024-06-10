{{/*
Expand the name of the chart.
*/}}
{{- define "spin-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "spin-operator.fullname" -}}
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
This chart may be used as part of a cloud provider marketplace offering.
Cloud providers may require certain values rules, for example Azure requires all images to flow through
via global.azure.images.
*/}}

{{/*
  Spin Operator image
*/}}
{{- define "spin-operator.manager.image" -}}
{{/*
  If Azure Marketplace, construct image from supplied values
*/}}
{{- if and
  .Values.global
  .Values.global.azure
  .Values.global.azure.images
  .Values.global.azure.images.spinOperator
  .Values.global.azure.images.spinOperator.registry
  .Values.global.azure.images.spinOperator.image
  .Values.global.azure.images.spinOperator.tag -}}
{{- printf "%s/%s:%s" .Values.global.azure.images.spinOperator.registry .Values.global.azure.images.spinOperator.image .Values.global.azure.images.spinOperator.tag }}
{{- else -}}
{{/*
  Use default values
*/}}
{{- printf "%s:%s" .Values.controllerManager.manager.image.repository (.Values.controllerManager.manager.image.tag | default .Chart.AppVersion) }}
{{- end }}
{{- end }}

{{/*
  RBAC proxy image
*/}}
{{- define "spin-operator.kubeRbacProxy.image" -}}
{{/*
  If Azure Marketplace, construct image from supplied values
*/}}
{{- if and
  .Values.global
  .Values.global.azure
  .Values.global.azure.images
  .Values.global.azure.images.kubeRbacProxy
  .Values.global.azure.images.kubeRbacProxy.registry
  .Values.global.azure.images.kubeRbacProxy.image
  .Values.global.azure.images.kubeRbacProxy.tag -}}
{{- printf "%s/%s:%s" .Values.global.azure.images.kubeRbacProxy.registry .Values.global.azure.images.kubeRbacProxy.image .Values.global.azure.images.kubeRbacProxy.tag }}
{{- else -}}
{{/*
  Use default values
*/}}
{{- printf "%s:%s" .Values.controllerManager.kubeRbacProxy.image.repository .Values.controllerManager.kubeRbacProxy.image.tag }}
{{- end }}
{{- end }}

{{/*
helmify replaces namespace name with `{{ .Release.Namespace }}` in dnsNames for Certificate object
which means `{{ include "spin-operator.fullname" . }}` gets replaced with `{{ include "{{ .Release.Namespace }}.fullname" . }}`

This is most likely a bug in helmify, but we can workaround it by defining a new template helper with name `{{ .Release.Namespace }}.fullname`
*/}}
{{- define "{{ .Release.Namespace }}.fullname" -}}
{{ include "spin-operator.fullname" . }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "spin-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "spin-operator.labels" -}}
helm.sh/chart: {{ include "spin-operator.chart" . }}
{{ include "spin-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "spin-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "spin-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "spin-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "spin-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
