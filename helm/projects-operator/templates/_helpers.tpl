{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "projects-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "projects-operator.fullname" -}}
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
{{- define "projects-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "projects-operator.labels" -}}
app.kubernetes.io/name: {{ include "projects-operator.name" . }}
helm.sh/chart: {{ include "projects-operator.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Generate client key and cert from CA
*/}}
{{- define "projects-operator.gen-webhook-certs" -}}
{{- $ca :=  genCA "projects-operator-ca" 365 -}}
{{- $cert := genSignedCert ( printf "%s-webhook.%s.svc" (include "projects-operator.fullname" .) .Release.Namespace ) nil nil 365 $ca -}}
caCert: {{ $ca.Cert | b64enc }}
clientCert: {{ $cert.Cert | b64enc }}
clientKey: {{ $cert.Key | b64enc }}
{{- end -}}

{{/*
Create a registry image reference for use in a spec.
Includes the `image` and `imagePullPolicy` keys.
*/}}
{{- define "projects-operator.registryImage" -}}
image: "{{ include "projects-operator.imageReference" . }}"
{{- $pullPolicy := include "projects-operator.imagePullPolicy" . -}}
{{- if $pullPolicy }}
{{ $pullPolicy }}
{{- end -}}
{{- end -}}
{{- define "projects-operator.imageReference" -}}
    {{- $registry := include "projects-operator.imageRegistry" . -}}
    {{- $namespace := include "projects-operator.imageNamespace" . -}}
    {{- printf "%s/%s/%s" $registry $namespace .image.name -}}
    {{- if .image.tag -}}
        {{- printf ":%s" .image.tag -}}
    {{- end -}}
{{- end -}}

{{- define "projects-operator.imageRegistry" -}}
    {{- if or (and .image.useOriginalRegistry (empty .image.registry)) (and .values.useOriginalRegistry (empty .values.imageRegistry)) -}}
        {{- include "projects-operator.originalImageRegistry" . -}}
    {{- else -}}
        {{- include "projects-operator.customImageRegistry" . -}}
    {{- end -}}
{{- end -}}

{{- define "projects-operator.originalImageRegistry" -}}
    {{- printf (coalesce .image.originalRegistry .values.originalImageRegistry "docker.io") -}}
{{- end -}}

{{- define "projects-operator.customImageRegistry" -}}
    {{- printf (coalesce .image.registry .values.imageRegistry .values.global.imageRegistry (include "projects-operator.originalImageRegistry" .)) -}}
{{- end -}}

{{- define "projects-operator.imageNamespace" -}}
    {{- if or (and .image.useOriginalNamespace (empty .image.namespace)) (and .values.useOriginalNamespace (empty .values.imageNamespace)) -}}
        {{- include "projects-operator.originalImageNamespace" . -}}
    {{- else -}}
        {{- include "projects-operator.customImageNamespace" . -}}
    {{- end -}}
{{- end -}}

{{- define "projects-operator.originalImageNamespace" -}}
    {{- printf (coalesce .image.originalNamespace .values.originalImageNamespace "bitnami") -}}
{{- end -}}

{{- define "projects-operator.customImageNamespace" -}}
    {{- printf (coalesce .image.namespace .values.imageNamespace .values.global.imageNamespace (include "projects-operator.originalImageNamespace" .)) -}}
{{- end -}}

{{/*
Specify the image pull policy
*/}}
{{- define "projects-operator.imagePullPolicy" -}}
    {{- $policy := coalesce .image.pullPolicy .values.global.imagePullPolicy -}}
    {{- if $policy -}}
        imagePullPolicy: "{{- $policy -}}"
    {{- end -}}
{{- end -}}

{{/*
Use the image pull secrets. All of the specified secrets will be used
*/}}
{{- define "projects-operator.imagePullSecrets" -}}
    {{- $secrets := .Values.global.imagePullSecrets -}}
    {{- range $_, $chartSecret := .Values.imagePullSecrets -}}
        {{- if $secrets -}}
            {{- $secrets = append $secrets $chartSecret -}}
        {{- else -}}
            {{- $secrets = list $chartSecret -}}
        {{- end -}}
    {{- end -}}
    {{- range $_, $image := .Values.images -}}
        {{- range $_, $s := $image.pullSecrets -}}
            {{- if $secrets -}}
                {{- $secrets = append $secrets $s -}}
            {{- else -}}
                {{- $secrets = list $s -}}
            {{- end -}}
        {{- end -}}
    {{- end -}}
    {{- if $secrets }}
imagePullSecrets:
        {{- range $secrets }}
  - name: {{ . }}
        {{- end }}
    {{- end -}}
{{- end -}}