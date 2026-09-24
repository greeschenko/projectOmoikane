# Production image for the Omoikane backend services (Phase 33).
#
# One multi-stage build produces ALL service binaries in a single image; each
# Kubernetes Deployment runs its own binary via `command: ["/app/bin/<svc>"]`.
# The monolith cmd/api is RETIRED — the nine binaries are:
#   auth content media messages settings trash dashboard docs audit
#
# Compose dev uses Dockerfile.dev (air hot-reload); this file is for the
# Helm chart / CI artifacts.

FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /app/bin && CGO_ENABLED=0 go build -trimpath -o /app/bin ./cmd/...

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/bin /app/bin
# Media uploads mount point: a PV is attached here in k8s (charts/omoikane),
# ../backend:/app bind in compose.
RUN mkdir -p /app/uploads
WORKDIR /app
EXPOSE 8080
CMD ["/app/bin/auth"]