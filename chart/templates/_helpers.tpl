{{/*
Return the full name for the chart
*/}}
{{- define "nextdns-operator.fullname" -}}
{{- include "bjw-s.common.lib.chart.names.fullname" . -}}
{{- end -}}

{{/*
Return the chart name
*/}}
{{- define "nextdns-operator.name" -}}
{{- include "bjw-s.common.lib.chart.names.name" . -}}
{{- end -}}
