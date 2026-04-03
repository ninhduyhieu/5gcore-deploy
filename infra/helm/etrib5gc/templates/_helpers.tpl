{{- define "etrib5gc.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "etrib5gc.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "etrib5gc.labels" -}}
app.kubernetes.io/name: {{ include "etrib5gc.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "etrib5gc.nfLabels" -}}
app: {{ .nfName }}
plmnId: {{ .Values.global.plmnIdLabel }}
{{- end }}

{{- define "etrib5gc.nfSelector" -}}
app: {{ .nfName }}
plmnId: {{ .Values.global.plmnIdLabel }}
{{- end }}

{{- define "etrib5gc.syncWave" -}}
{{- if hasPrefix "0" .wave }}argocd.argoproj.io/sync-wave: "0"{{ end }}
{{- if hasPrefix "1" .wave }}argocd.argoproj.io/sync-wave: "1"{{ end }}
{{- if hasPrefix "2" .wave }}argocd.argoproj.io/sync-wave: "2"{{ end }}
{{- if hasPrefix "3" .wave }}argocd.argoproj.io/sync-wave: "3"{{ end }}
{{- if hasPrefix "4" .wave }}argocd.argoproj.io/sync-wave: "4"{{ end }}
{{- end }}
