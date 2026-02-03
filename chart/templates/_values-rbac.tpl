{{/*
RBAC configuration for nextdns-operator
AUTO-GENERATED from config/rbac/role.yaml - DO NOT EDIT MANUALLY
Run 'make generate-helm-rbac' to regenerate after updating kubebuilder markers.
*/}}
{{- define "nextdns-operator.values.rbac" -}}
{{- if .Values.rbac.enabled }}
rbac:
  roles:
    main:
      enabled: true
      type: ClusterRole
      rules:
        - apiGroups:
            - ""
          resources:
            - configmaps
            - services
          verbs:
            - create
            - delete
            - get
            - list
            - patch
            - update
            - watch
        - apiGroups:
            - ""
          resources:
            - secrets
          verbs:
            - get
            - list
            - watch
        - apiGroups:
            - apps
          resources:
            - daemonsets
            - deployments
          verbs:
            - create
            - delete
            - get
            - list
            - patch
            - update
            - watch
        - apiGroups:
            - coordination.k8s.io
          resources:
            - leases
          verbs:
            - create
            - delete
            - get
            - list
            - patch
            - update
            - watch
        - apiGroups:
            - nextdns.io
          resources:
            - nextdnsallowlists
            - nextdnscoredns
            - nextdnsdenylists
            - nextdnsprofiles
            - nextdnstldlists
          verbs:
            - create
            - delete
            - get
            - list
            - patch
            - update
            - watch
        - apiGroups:
            - nextdns.io
          resources:
            - nextdnsallowlists/finalizers
            - nextdnscoredns/finalizers
            - nextdnsdenylists/finalizers
            - nextdnsprofiles/finalizers
            - nextdnstldlists/finalizers
          verbs:
            - update
        - apiGroups:
            - nextdns.io
          resources:
            - nextdnsallowlists/status
            - nextdnscoredns/status
            - nextdnsdenylists/status
            - nextdnsprofiles/status
            - nextdnstldlists/status
          verbs:
            - get
            - patch
            - update
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
