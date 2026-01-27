{{/*
Build defaultPodOptions structure from flat values
*/}}
{{- define "nextdns-operator.values.security" -}}
defaultPodOptions:
  enableServiceLinks: false
  hostIPC: false
  hostNetwork: false
  hostPID: false
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
{{- end -}}
