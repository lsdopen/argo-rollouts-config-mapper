{{/*
Expand the name of the chart.
*/}}
{{- define "argo-rollouts-config-mapper.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "argo-rollouts-config-mapper.fullname" -}}
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
{{- define "argo-rollouts-config-mapper.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "argo-rollouts-config-mapper.labels" -}}
helm.sh/chart: {{ include "argo-rollouts-config-mapper.chart" . }}
{{ include "argo-rollouts-config-mapper.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.labels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "argo-rollouts-config-mapper.selectorLabels" -}}
app.kubernetes.io/name: {{ include "argo-rollouts-config-mapper.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "argo-rollouts-config-mapper.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "argo-rollouts-config-mapper.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Validate required values
*/}}
{{- define "argo-rollouts-config-mapper.validateValues" -}}
{{- if not .Values.certificates.method }}
{{- fail "certificates.method is required (helm, cert-manager, or external)" }}
{{- end }}
{{- if eq .Values.certificates.method "helm" }}
{{- if not .Values.certificates.helm.duration }}
{{- fail "certificates.helm.duration is required when using helm certificate method" }}
{{- end }}
{{- if not .Values.certificates.helm.subject.organizationName }}
{{- fail "certificates.helm.subject.organizationName is required when using helm certificate method" }}
{{- end }}
{{- else if eq .Values.certificates.method "cert-manager" }}
{{- if not .Values.certificates.certManager.issuer.name }}
{{- fail "certificates.certManager.issuer.name is required when using cert-manager certificate method" }}
{{- end }}
{{- if not .Values.certificates.certManager.issuer.kind }}
{{- fail "certificates.certManager.issuer.kind is required when using cert-manager certificate method" }}
{{- end }}
{{- else if eq .Values.certificates.method "external" }}
{{- if not .Values.certificates.external.secretName }}
{{- fail "certificates.external.secretName is required when using external certificate method" }}
{{- end }}
{{- if not .Values.certificates.external.certFile }}
{{- fail "certificates.external.certFile is required when using external certificate method" }}
{{- end }}
{{- if not .Values.certificates.external.keyFile }}
{{- fail "certificates.external.keyFile is required when using external certificate method" }}
{{- end }}
{{- if not .Values.certificates.external.caBundle }}
{{- fail "certificates.external.caBundle is required when using external certificate method" }}
{{- end }}
{{- else }}
{{- fail "certificates.method must be one of: helm, cert-manager, external" }}
{{- end }}
{{- end }}

{{/*
Get the certificate secret name
*/}}
{{- define "argo-rollouts-config-mapper.certificateSecretName" -}}
{{- if eq .Values.certificates.method "external" }}
{{- .Values.certificates.external.secretName }}
{{- else }}
{{- printf "%s-certs" (include "argo-rollouts-config-mapper.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Get the certificate file name
*/}}
{{- define "argo-rollouts-config-mapper.certificateFile" -}}
{{- if eq .Values.certificates.method "external" }}
{{- .Values.certificates.external.certFile }}
{{- else }}
{{- "tls.crt" }}
{{- end }}
{{- end }}

{{/*
Get the private key file name
*/}}
{{- define "argo-rollouts-config-mapper.privateKeyFile" -}}
{{- if eq .Values.certificates.method "external" }}
{{- .Values.certificates.external.keyFile }}
{{- else }}
{{- "tls.key" }}
{{- end }}
{{- end }}

{{/*
Get the CA bundle
*/}}
{{- define "argo-rollouts-config-mapper.caBundle" -}}
{{- if eq .Values.certificates.method "external" }}
{{- .Values.certificates.external.caBundle }}
{{- else if eq .Values.certificates.method "helm" }}
{{- $ca := genCA (printf "%s-ca" (include "argo-rollouts-config-mapper.fullname" .)) (.Values.certificates.helm.duration | int) }}
{{- $cert := genSignedCert (include "argo-rollouts-config-mapper.serviceName" .) nil (list (include "argo-rollouts-config-mapper.serviceName" .) (printf "%s.%s" (include "argo-rollouts-config-mapper.serviceName" .) .Release.Namespace) (printf "%s.%s.svc" (include "argo-rollouts-config-mapper.serviceName" .) .Release.Namespace) (printf "%s.%s.svc.cluster.local" (include "argo-rollouts-config-mapper.serviceName" .) .Release.Namespace)) (.Values.certificates.helm.duration | int) $ca }}
{{- $ca.Cert | b64enc }}
{{- else }}
{{- "" }}
{{- end }}
{{- end }}

{{/*
Get the service name for certificates
*/}}
{{- define "argo-rollouts-config-mapper.serviceName" -}}
{{- include "argo-rollouts-config-mapper.fullname" . }}
{{- end }}
