{{- define "etrib5gc.image" -}}
{{- printf "%s/%s:latest" .Values.global.imageRegistry .image -}}
{{- end -}}
