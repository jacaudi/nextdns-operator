{{/*
Build serviceAccount structure from flat values
*/}}
{{- define "nextdns-operator.values.serviceaccount" -}}
serviceAccount:
  create: {{ .Values.serviceAccount.create }}
  {{- if .Values.serviceAccount.name }}
  name: {{ .Values.serviceAccount.name }}
  {{- end }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end -}}
