# Omoikane on Microsoft Azure AKS — runbook (Phase 36)

Deploy target: AKS + Azure Database for PostgreSQL Flexible Server + Confluent
Cloud Kafka + Azure Cache for Redis. Azure has no native managed Kafka —
Confluent Cloud (SASL_SSL/PLAIN) is the runbook default; Aiven/Redpanda Cloud or
the in-cluster KRaft broker are equivalent alternatives.

## 1. Prerequisites

- `az` CLI, `az aks create`/existing AKS cluster (`az aks get-credentials`),
  `kubectl`, `helm` ≥ v4 (CI pins v4.3.0 — the migration-hook document-separator semantics were verified on v4)
- GitHub repo with `deploy.yml`; `environment: production` with
  `DEPLOY_KUBECONFIG`/`DEPLOY_VALUES`

## 2. Managed services

```bash
# Azure Database for PostgreSQL — Flexible Server
az postgres flexible-server create --name omoikane-db \
  --resource-group omoikane-rg --admin-user omoikane --admin-password 'CHANGE-ME' \
  --public-access 0.0.0.0/0     # or --vnet <aks-vnet> for private access
PG_HOST='omoikane-db.postgres.database.azure.com'
az postgres flexible-server db create -g omoikane-rg -s omoikane-db -d omoikane
az postgres flexible-server db create -g omoikane-rg -s omoikane-db -d omoikane_audit

# Azure Cache for Redis
az redis create --name omoikane-redis --resource-group omoikane-rg \
  --sku Basic --vm-size c0 --enable-non-ssl-port
REDIS_HOST=$(az redis show --name omoikane-redis -g omoikane-rg \
  --query 'hostName' --output tsv)
REDIS_KEY=$(az redis list-keys --name omoikane-redis -g omoikane-rg \
  --query 'primaryKey' --output tsv)

# Confluent Cloud Kafka (SASL_SSL/PLAIN) — confluent CLI creates a cluster in
# an Azure region; the API key/secret pair is the SASL credential.
CC_BOOTSTRAP='pkc-xxxxx.eastus.azure.confluent.cloud:9092'
```

## 3. Install

`charts/omoikane/values-aks.yaml` carries the endpoint skeleton (fill the
`CHANGE-ME`s). Credentials via a gitignored `deploy-override.yaml` or `--set`:

```bash
helm upgrade --install omoikane charts/omoikane \
  -f charts/omoikane/values-aks.yaml \
  -f /tmp/deploy-override.yaml \
  --namespace omoikane --create-namespace \
  --set external.postgres.host="$PG_HOST" \
  --set external.redis.url="rediss://:$REDIS_KEY@$REDIS_HOST:6380/0" \
  --set external.kafka.bootstrap="$CC_BOOTSTRAP" \
  --wait --timeout 15m
```

`deploy-override.yaml`:

```yaml
external:
  postgres:
    password: '<flexible-server-password>'
  kafka:
    saslUsername: '<confluent-api-key>'
secrets:
  jwtSecret: '<random>'
  internalToken: '<random>'
  externalKafkaPassword: '<confluent-api-secret>'
```

Notes:

- Azure enforces TLS (`sslmode=require` preset in `values-aks.yaml`) and
  password-based auth only — the DSN comes straight from `external.postgres`.
- Azure Cache default is TLS on port 6380 (`rediss://`); the non-SSL variant
  (`redis://...:6379/0`) works when `--enable-non-ssl-port` is set.
- Prefer VNet-integrated PostgreSQL/Redis for anything beyond a demo.

## 4. DNS/TLS (optional)

```bash
kubectl -n omoikane get svc nginx -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Enable `ingress.enabled` + cert-manager for TLS in front of nginx.

## 5. FIRST RESULT

```bash
scripts/cloud-smoke.sh "http://<lb-ip>"
```

Must end with `FIRST RESULT verified`. See `docs/cloud/verify-first-result.md`.