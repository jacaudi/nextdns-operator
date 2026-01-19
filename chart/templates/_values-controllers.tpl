{{/*
Build controllers structure from flat values
*/}}
{{- define "nextdns-operator.values.controllers" -}}
controllers:
  main:
    type: deployment
    replicas: {{ .Values.controller.replicas }}
    strategy: {{ .Values.controller.strategy }}
    containers:
      main:
        image:
          repository: {{ .Values.image.repository }}
          tag: {{ .Values.image.tag }}
          pullPolicy: {{ .Values.image.pullPolicy }}
        args:
          - --leader-elect
          - --health-probe-bind-address=:8081
          - --metrics-bind-address=:8080
        env:
          TZ: {{ .Values.timezone }}
        resources:
          {{- toYaml .Values.resources | nindent 10 }}
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
              - ALL
        probes:
          liveness:
            enabled: true
            custom: true
            spec:
              httpGet:
                path: /healthz
                port: 8081
              initialDelaySeconds: 15
              periodSeconds: 20
          readiness:
            enabled: true
            custom: true
            spec:
              httpGet:
                path: /readyz
                port: 8081
              initialDelaySeconds: 5
              periodSeconds: 10
{{- end -}}
