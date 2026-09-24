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
*/}}
{{- define "omoikane.postgresDSN" -}}
{{- $ctx := . -}}
host=postgres port=5432 user={{ $ctx.Values.postgres.user }} password={{ $ctx.Values.postgres.password }} dbname={{ index $ctx.Values.backend.databases $ctx.svc }} sslmode=disable
{{- end -}}