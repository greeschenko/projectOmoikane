# Omoikane on Amazon EKS — runbook (Phase 36)

One-button FAAST deploy target: EKS + RDS PostgreSQL + MSK + ElastiCache.
The chart is cloud-agnostic; this runbook only wires the managed endpoints.

## 1. Prerequisites

- `aws` CLI configured, `eksctl` (or console) access, `kubectl`, `helm` ≥ v4 (CI pins v4.3.0 — the migration-hook document-separator semantics were verified on v4)
- Existing VPC; EKS cluster (e.g. `eksctl create cluster --name omoikane`)
- GitHub repo with the Phase 36 workflow + `deploy.yml`; `environment:
  production` configured with `DEPLOY_KUBECONFIG`/`DEPLOY_VALUES`

## 2. Managed services

```bash
# RDS PostgreSQL (both target databases must exist)
aws rds create-db-instance \
  --db-instance-identifier omoikane \
  --engine postgres --engine-version 16 \
  --db-instance-class db.t3.medium \
  --master-username omoikane --master-user-password 'CHANGE-ME' \
  --allocated-storage 20 --publicly-accessible false \
  --vpc-security-group-ids sg-...   # allow 5432 from the EKS node SG
RDS_HOST=$(aws rds describe-db-instances --db-instance-identifier omoikane \
  --query 'DBInstances[0].Endpoint.Address' --output text)

# MSK (in-VPC PLAINTEXT is the default here; SASL for stricter setups — see below)
aws kafka create-cluster-v2 --cluster-name omoikane-msk \
  --provisioned '
    {
      "brokerNodeGroupInfo": { "instanceType": "kafka.m5.large",
        "clientSubnets": ["subnet-..."], "securityGroups": ["sg-..."] },
      "numberOfBrokerNodes": 3,
      "kafkaVersion": "3.7.0"
    }'
MSK_BOOTSTRAP=$(aws kafka get-bootstrap-brokers --cluster-arn <arn> \
  --query 'BootstrapBrokerString' --output text)

# ElastiCache Redis
aws elasticache create-cache-cluster --cache-cluster-id omoikane-redis \
  --engine redis --cache-node-type cache.t3.micro --num-cache-nodes 1
REDIS_HOST=$(aws elasticache describe-cache-clusters --cache-cluster-id omoikane-redis \
  --show-cache-node-info --query 'CacheClusters[0].CacheNodes[0].Endpoint.Address' --output text)
```

Pre-create both databases; services self-migrate at startup but the DSNs must
resolve:

```bash
psql "postgres://omoikane:<pw>@$RDS_HOST:5432/postgres" \
  -c 'CREATE DATABASE omoikane;' -c 'CREATE DATABASE omoikane_audit;'
```

MSK disables auto-create — let `EnsureTopics` provision on startup (needs
`CreateTopic` on the IAM role) or provision the two topics up front:

```bash
/opt/aws/msk/.../bin/kafka-topics.sh --bootstrap-server "$MSK_BOOTSTRAP" \
  --create --topic omoikane.events --partitions 3 --replication-factor 3
# ... and omoikane.events.dlq
```

## 3. Install

`charts/omoikane/values-eks.yaml` carries the endpoint skeleton (edit the
`CHANGE-ME` placeholders). Credentials are passed separately — a gitignored
drop-in `deploy-override.yaml` or `--set`:

```bash
helm upgrade --install omoikane charts/omoikane \
  -f charts/omoikane/values-eks.yaml \
  -f /tmp/deploy-override.yaml \
  --namespace omoikane --create-namespace \
  --set external.postgres.host="$RDS_HOST" \
  --set images.backend.repository=ghcr.io/<owner>/omoikane-backend \
  --set images.backend.tag=phase36 \
  --set images.frontend.repository=ghcr.io/<owner>/omoikane-frontend \
  --set images.frontend.tag=phase36 \
  --wait --timeout 15m
```

with `deploy-override.yaml`:

```yaml
external:
  postgres:
    password: '<rds-password>'
  kafka:
    saslUsername: '<msk-user>'      # only with SASL_SSL auth
secrets:
  jwtSecret: '<random>'
  internalToken: '<random>'
  externalKafkaPassword: '<msk-sasl-secret>'   # only with SASL_SSL auth
```

(The GitHub `deploy` job does exactly this via its `DEPLOY_VALUES` secret.)

Prefer SASL for MSK: set `external.kafka.securityProtocol: SASL_SSL`,
`saslMechanism: SCRAM-SHA-512` and enable SASL/SCRAM on the cluster.

## 4. DNS/TLS (optional)

The gateway Service is a classic LoadBalancer (`aws elbv2`):

```bash
kubectl -n omoikane get svc nginx -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
```

Point a domain at it, then enable `ingress.enabled` + cert-manager
(`ingress.host: omoikane.example.com`) if you want TLS in front of nginx.

## 5. FIRST RESULT

```bash
scripts/cloud-smoke.sh "http://<elb-hostname>"
```

Must end with `FIRST RESULT verified`. See `docs/cloud/verify-first-result.md`.