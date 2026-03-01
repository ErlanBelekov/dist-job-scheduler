---
name: k8s
description: Manage the dist-scheduler k3s cluster. Use when the user asks about pods, deployments, logs, cluster status, or wants to run kubectl commands.
allowed-tools: Bash, Read, Grep, Edit
---

# Kubernetes Cluster Management

You are managing a k3s cluster for the dist-job-scheduler project.

## Cluster Details

- **Server IP:** 91.99.129.152
- **Kubeconfig:** `~/.kube/dist-scheduler.yaml`
- **Namespace:** `dist-scheduler`
- **Domain:** `job.enkiduck.com`

## Environment

All kubectl commands MUST use:
```
KUBECONFIG=~/.kube/dist-scheduler.yaml kubectl <command> -n dist-scheduler
```

Shorthand for this skill — define at the start of any bash session:
```bash
K="KUBECONFIG=~/.kube/dist-scheduler.yaml"
```

## Workloads

| Workload | Type | Replicas | Notes |
|----------|------|----------|-------|
| postgres | StatefulSet | 1 | PostgreSQL 16, headless service, 5Gi PVC |
| server | Deployment | 2 | HTTP API on port 8080, ClusterIP service |
| scheduler | Deployment | 1 | Worker + Reaper, never scale beyond 1 |
| migrate | Job | one-off | Runs goose migrations |

## Common Operations

When the user asks to: **$ARGUMENTS**

### Status Check
```bash
kubectl get pods -n dist-scheduler
kubectl get svc,ingress -n dist-scheduler
```

### View Logs
```bash
kubectl logs deployment/server -n dist-scheduler --tail=50
kubectl logs deployment/scheduler -n dist-scheduler --tail=50
kubectl logs job/migrate -n dist-scheduler
```

### Restart a Deployment
```bash
kubectl rollout restart deployment/<name> -n dist-scheduler
kubectl rollout status deployment/<name> -n dist-scheduler --timeout=120s
```

### Re-run Migrations
```bash
kubectl delete job migrate -n dist-scheduler --ignore-not-found
kubectl apply -f infra/k8s/migrate-job.yaml
kubectl wait --for=condition=Complete job/migrate -n dist-scheduler --timeout=60s
```

### Update Secrets (after editing infra/k8s/secrets.yaml)
```bash
kubectl apply -f infra/k8s/secrets.yaml
kubectl rollout restart deployment/server deployment/scheduler -n dist-scheduler
```

### Update ConfigMap (after editing infra/k8s/configmap.yaml)
```bash
kubectl apply -f infra/k8s/configmap.yaml
kubectl rollout restart deployment/server deployment/scheduler -n dist-scheduler
```

### Debug CrashLoopBackOff
1. Check logs: `kubectl logs <pod-name> -n dist-scheduler --previous`
2. Check events: `kubectl describe pod <pod-name> -n dist-scheduler`
3. Check secret/configmap are correct: `kubectl get secret dist-scheduler -n dist-scheduler -o yaml`

### Port Forward (for local access)
```bash
kubectl port-forward pod/postgres-0 -n dist-scheduler 5432:5432
kubectl port-forward deployment/server -n dist-scheduler 8080:8080
```

### Check TLS Certificate
```bash
kubectl get certificate -n dist-scheduler
kubectl describe certificate server-tls -n dist-scheduler
```

## Manifest Locations

- `infra/k8s/namespace.yaml`
- `infra/k8s/secrets.yaml` — contains DB creds, JWT_SECRET, Resend config
- `infra/k8s/configmap.yaml` — ENV, PORT, WORKER_COUNT, POLL_INTERVAL_SEC, MAGIC_LINK_BASE_URL
- `infra/k8s/postgres.yaml` — StatefulSet + headless Service + PVC
- `infra/k8s/migrate-job.yaml` — goose migration Job
- `infra/k8s/server.yaml` — Deployment + ClusterIP Service
- `infra/k8s/scheduler.yaml` — Deployment (Recreate strategy)
- `infra/k8s/ingress.yaml` — Traefik ingress for job.enkiduck.com
- `infra/k8s/cert-manager.yaml` — Let's Encrypt ClusterIssuer

## Container Registry

Images are stored in GCP Artifact Registry:
```
europe-west3-docker.pkg.dev/notifylm/dist-scheduler/{server,scheduler,migrate}:latest
```

## Important Rules

1. NEVER delete the postgres StatefulSet or its PVC without explicit user confirmation — this destroys data
2. The scheduler must NEVER have more than 1 replica
3. Always use `--ignore-not-found` when deleting jobs before re-creating them
4. When troubleshooting, always check logs FIRST before suggesting changes
