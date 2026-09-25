{{/*
Common labels for all Omoikane resources.
*/}}
{{- define "omoikane.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/part-of: omoikane
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{/*
Selector labels for deployments/services. Kept minimal (a single `app` label)
so probes/selectors stay predictable; the richer set lives in metadata.labels.
*/}}
{{- define "omoikane.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
The postgres DSN template used by every database-backed service.
Context: dict {Values: <chart values>, svc: <service name key>}.

With external.postgres.enabled the DSN points at the managed endpoint
(RDS/CloudSQL/Azure DB) — the same DB-name map (.Values.backend.databases)
still applies, so both `omoikane` and `omoikane_audit` must exist there.
*/}}
{{- define "omoikane.postgresDSN" -}}
{{- $ctx := . -}}
{{- if $ctx.Values.external.postgres.enabled -}}
host={{ $ctx.Values.external.postgres.host }} port={{ $ctx.Values.external.postgres.port }} user={{ $ctx.Values.external.postgres.user }} password={{ $ctx.Values.external.postgres.password }} dbname={{ index $ctx.Values.backend.databases $ctx.svc }} sslmode={{ $ctx.Values.external.postgres.sslmode }}
{{- else -}}
host=postgres port=5432 user={{ $ctx.Values.postgres.user }} password={{ $ctx.Values.postgres.password }} dbname={{ index $ctx.Values.backend.databases $ctx.svc }} sslmode=disable
{{- end -}}
{{- end -}}

{{/*
The Kafka bootstrap the services talk to: the internal KRaft broker by default,
or the managed endpoint (MSK/Confluent Cloud) when external.kafka is enabled.
*/}}
{{- define "omoikane.kafkaBootstrap" -}}
{{- if .Values.external.kafka.enabled }}{{ .Values.external.kafka.bootstrap }}{{ else }}{{ .Values.kafka.bootstrap }}{{ end -}}
{{- end -}}

{{/*
The shared-cache REDIS_URL: the in-cluster redis service by default, or the
managed cache endpoint (ElastiCache/Redis Cloud) when external.redis is enabled.
*/}}
{{- define "omoikane.redisURL" -}}
{{- if .Values.external.redis.enabled }}{{ .Values.external.redis.url }}{{ else }}redis://redis:6379/0{{ end -}}
{{- end -}}