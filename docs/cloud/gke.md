# Omoikane on Google Cloud GKE — runbook (Phase 36)

Deploy target: GKE + Cloud SQL PostgreSQL + Confluent Cloud Kafka + Memorystore
Redis. GCP has no native managed Kafka — the runbook's default is Confluent
Cloud (SASL_SSL/PLAIN); Aiven/Redpanda Cloud or the in-cluster KRaft broker are
equivalent alternatives.

## 1. Prerequisites

- `gcloud` configured, a GKE cluster
  (`gcloud container clusters create omoikane --zone us-central1-a --num-nodes 2`)
- GitHub repo with `deploy.yml`; `environment: production` with
  `DEPLOY_KUBECONFIG`/`DEPLOY_VALUES`

## 2. Managed services

```bash
# Cloud SQL for PostgreSQL
gcloud sql instances create omoikane-sql --database-version=POSTGRES_16 \
  --tier=db-f1-micro --region=us-central1 \
  --network=<vpc> --no-assign-ip        # private IP; or --assign-ip + authorized nets for demo
gcloud sql users create omoikane --instance=omoikane-sql \
  --password='CHANGE-ME'
SQL_IP=$(gcloud sql instances describe omoikane-sql \
  --format='value(ipAddresses[0].ipAddress)')
gcloud sql databases create omoikane      --instance=omoikane-sql
gcloud sql databases create omoikane_audit --instance=omoikane-sql

# Confluent Cloud Kafka (SASL_SSL/PLAIN). `confluent` CLI:
#   confluent kafka cluster create omoikane --cloud gcp --region us-central1 ...
#   confluent kafka api-key create --resource <cluster>          # -> key + secret
CC_BOOTSTRAP='pkc-xxxxx.us-central1.gcp.confluent.cloud:9092'

# Memorystore Redis
gcloud redis instances create omoikane-redis --size=1 --region=us-central1 \
  --network=<vpc> --connect-mode=PRIVATE_SERVICE_ACCESS
REDIS_IP=$(gcloud redis instances describe omoikane-redis --region=us-central1 \
  --format='value(host)')
```

## 3. Install

`charts/omoikane/values-gke.yaml` carries the endpoint skeleton. Credentials are
passed via a gitignored `deploy-override.yaml` or `--set`:

```bash
helm upgrade --install omoikane charts/omoikane \
  -f charts/omoikane/values-gke.yaml \
  -f /tmp/deploy-override.yaml \
  --namespace omoikane --create-namespace \
  --set external.postgres.host="$SQL_IP" \
  --set external.redis.url="redis://$REDIS_IP:6379/0" \
  --set external.kafka.bootstrap="$CC_BOOTSTRAP" \
  --wait --timeout 15m
```

`deploy-override.yaml`:

```yaml
external:
  postgres:
    password: '<cloudsql-password>'
  kafka:
    saslUsername: '<confluent-api-key>'
secrets:
  jwtSecret: '<random>'
  internalToken: '<random>'
  externalKafkaPassword: '<confluent-api-secret>'
```

Notes:

- Cloud SQL `sslmode=require` is preset in `values-gke.yaml`; the Go driver
  supports it out of the box.
- VPC peering/Private Service Connect must be in place before the install so
  pods can reach the Cloud SQL + Memorystore private IPs.
- Confluent Cloud pre-provisions topics; `EnsureTopics` at service startup is a
  no-op on existing names (auto-create can stay on there).

## 4. DNS/TLS (optional)

```bash
kubectl -n omoikane get svc nginx -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Enable `ingress.enabled` + a Google-managed cert (cert-manager) for TLS.

## 5. FIRST RESULT

```bash
scripts/cloud-smoke.sh "http://<lb-ip>"
```

Must end with `FIRST RESULT verified`. See `docs/cloud/verify-first-result.md`.