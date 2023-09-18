{{/*
Expand the name of the chart.
*/}}
{{- define "kubeenforcer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "kubeenforcer.admission-controller.name" -}}
{{ template "kubeenforcer.name" . }}-admission-controller
{{- end -}}

{{- define "kubeenforcer.admission-controller.serviceName" -}}
{{- printf "%s-svc" (include "kubeenforcer.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubeenforcer.fullname" -}}
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
{{- define "kubeenforcer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "kubeenforcer.namespace" -}}
{{ default .Release.Namespace .Values.namespaceOverride }}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "kubeenforcer.labels" -}}
helm.sh/chart: {{ include "kubeenforcer.chart" . }}
{{ include "kubeenforcer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kubeenforcer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubeenforcer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kubeenforcer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kubeenforcer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the cluster role to use
*/}}
{{- define "kubeenforcer.clusterRoleName" -}}
{{- default (include "kubeenforcer.fullname" .) .Values.rbac.clusterRoleName }}
{{- end }}

{{/*
Create the name of the cluster role binding to use
*/}}
{{- define "kubeenforcer.clusterRoleBindingName" -}}
{{- default (include "kubeenforcer.fullname" .) .Values.rbac.clusterRoleBindingName }}
{{- end }}


