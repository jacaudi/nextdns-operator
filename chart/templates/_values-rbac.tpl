{{/*
Build RBAC structure from flat values
*/}}
{{- define "nextdns-operator.values.rbac" -}}
{{- if .Values.rbac.enabled }}
rbac:
  roles:
    main:
      enabled: true
      type: ClusterRole
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
        # NextDNS List CRD permissions
        - apiGroups: ["nextdns.io"]
          resources: ["nextdnsallowlists", "nextdnsdenylists", "nextdnstldlists"]
          verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
        - apiGroups: ["nextdns.io"]
          resources: ["nextdnsallowlists/status", "nextdnsdenylists/status", "nextdnstldlists/status"]
          verbs: ["get", "patch", "update"]
        - apiGroups: ["nextdns.io"]
          resources: ["nextdnsallowlists/finalizers", "nextdnsdenylists/finalizers", "nextdnstldlists/finalizers"]
          verbs: ["update"]
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
  bindings:
    main:
      enabled: true
      type: ClusterRoleBinding
      roleRef:
        identifier: main
      subjects:
        - identifier: main
{{- end }}
{{- end -}}
