{{/*
Build RBAC structure from flat values
*/}}
{{- define "nextdns-operator.values.rbac" -}}
{{- if .Values.rbac.enabled }}
rbac:
  enabled: true
  rules:
    # NextDNS Profile CRD permissions
    - apiGroups: ["nextdns.io"]
      resources: ["nextdnsprofiles"]
      verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
    - apiGroups: ["nextdns.io"]
      resources: ["nextdnsprofiles/status"]
      verbs: ["get", "patch", "update"]
    - apiGroups: ["nextdns.io"]
      resources: ["nextdnsprofiles/finalizers"]
      verbs: ["update"]
    # NextDNS List CRD permissions (read-only)
    - apiGroups: ["nextdns.io"]
      resources: ["nextdnsallowlists", "nextdnsdenylists", "nextdnstldlists"]
      verbs: ["get", "list", "watch"]
    # Secret access for API credentials
    - apiGroups: [""]
      resources: ["secrets"]
      verbs: ["get", "list", "watch"]
    # Events for status reporting
    - apiGroups: [""]
      resources: ["events"]
      verbs: ["create", "patch"]
    # Leader election
    - apiGroups: ["coordination.k8s.io"]
      resources: ["leases"]
      verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
{{- end }}
{{- end -}}
