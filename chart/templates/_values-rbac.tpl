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
            - nextdns.io
          resources:
            - nextdnsallowlists
            - nextdnsdenylists
            - nextdnstldlists
          verbs:
            - get
            - list
            - watch
        - apiGroups:
            - nextdns.io
          resources:
            - nextdnsprofiles
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
            - nextdnsprofiles/finalizers
          verbs:
            - update
        - apiGroups:
            - nextdns.io
          resources:
            - nextdnsprofiles/status
          verbs:
            - get
            - patch
            - update
        - apiGroups:
            - nextdns.jacaudi.com
          resources:
            - nextdnsallowlists
            - nextdnsdenylists
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
            - nextdns.jacaudi.com
          resources:
            - nextdnsallowlists/finalizers
            - nextdnsdenylists/finalizers
            - nextdnstldlists/finalizers
          verbs:
            - update
        - apiGroups:
            - nextdns.jacaudi.com
          resources:
            - nextdnsallowlists/status
            - nextdnsdenylists/status
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
