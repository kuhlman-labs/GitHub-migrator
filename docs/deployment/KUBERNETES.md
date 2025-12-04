# Kubernetes Deployment

This guide covers deploying GitHub Migrator to Kubernetes clusters.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Basic Deployment](#basic-deployment)
- [Production Configuration](#production-configuration)
- [Ingress and TLS](#ingress-and-tls)
- [PostgreSQL Setup](#postgresql-setup)
- [Scaling](#scaling)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Overview

This deployment creates:
- Deployment with configurable replicas
- Service for internal and external access
- ConfigMap for application configuration
- Secrets for sensitive data
- Persistent volumes for data and logs
- Optional Ingress with TLS termination

## Prerequisites

- Kubernetes cluster (1.21+)
- kubectl configured for your cluster
- Docker image built and pushed to a registry
- Persistent storage provisioner (for production)

## Basic Deployment

### 1. Create Namespace

```bash
kubectl create namespace github-migrator
```

### 2. Create Secrets

```bash
kubectl create secret generic github-tokens \
  --from-literal=source-token="${GITHUB_SOURCE_TOKEN}" \
  --from-literal=dest-token="${GITHUB_DEST_TOKEN}" \
  -n github-migrator
```

### 3. Create ConfigMap

Create `configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migrator-config
  namespace: github-migrator
data:
  config.yaml: |
    server:
      port: 8080
    database:
      type: sqlite
      dsn: /app/data/migrator.db
    github:
      source:
        base_url: "https://github.company.com/api/v3"
      destination:
        base_url: "https://api.github.com"
    logging:
      level: info
      format: json
      output_file: /app/logs/migrator.log
```

```bash
kubectl apply -f configmap.yaml
```

### 4. Create Persistent Volume Claims

Create `pvc.yaml`:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: migrator-data
  namespace: github-migrator
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: migrator-logs
  namespace: github-migrator
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

```bash
kubectl apply -f pvc.yaml
```

### 5. Create Deployment

Create `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: github-migrator
  namespace: github-migrator
  labels:
    app: github-migrator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: github-migrator
  template:
    metadata:
      labels:
        app: github-migrator
    spec:
      containers:
      - name: migrator
        image: ghcr.io/your-org/github-migrator:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: GITHUB_SOURCE_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-tokens
              key: source-token
        - name: GITHUB_DEST_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-tokens
              key: dest-token
        - name: GHMIG_SERVER_PORT
          value: "8080"
        - name: GHMIG_DATABASE_TYPE
          value: "sqlite"
        - name: GHMIG_DATABASE_DSN
          value: "/app/data/migrator.db"
        - name: GHMIG_LOGGING_LEVEL
          value: "info"
        volumeMounts:
        - name: data
          mountPath: /app/data
        - name: logs
          mountPath: /app/logs
        - name: config
          mountPath: /app/configs
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: migrator-data
      - name: logs
        persistentVolumeClaim:
          claimName: migrator-logs
      - name: config
        configMap:
          name: migrator-config
```

```bash
kubectl apply -f deployment.yaml
```

### 6. Create Service

Create `service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: github-migrator
  namespace: github-migrator
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: github-migrator
```

```bash
kubectl apply -f service.yaml
```

### 7. Verify Deployment

```bash
# Check pods
kubectl get pods -n github-migrator

# Check service
kubectl get svc -n github-migrator

# View logs
kubectl logs -f deployment/github-migrator -n github-migrator

# Port forward for testing
kubectl port-forward svc/github-migrator 8080:80 -n github-migrator
```

## Production Configuration

### High Availability

For production, increase replicas and use PostgreSQL:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: github-migrator
  namespace: github-migrator
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: github-migrator
              topologyKey: kubernetes.io/hostname
      containers:
      - name: migrator
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

### Resource Limits by Scale

| Scale | Memory Request | Memory Limit | CPU Request | CPU Limit |
|-------|---------------|--------------|-------------|-----------|
| Small (<1k repos) | 256Mi | 512Mi | 100m | 500m |
| Medium (1k-10k repos) | 512Mi | 1Gi | 250m | 1000m |
| Large (10k+ repos) | 1Gi | 2Gi | 500m | 2000m |

## Ingress and TLS

### Nginx Ingress with cert-manager

Create `ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: github-migrator
  namespace: github-migrator
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: "0"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
spec:
  tls:
  - hosts:
    - migrator.company.com
    secretName: migrator-tls
  rules:
  - host: migrator.company.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: github-migrator
            port:
              number: 80
```

```bash
kubectl apply -f ingress.yaml
```

### LoadBalancer Service (Alternative)

For cloud providers without Ingress:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: github-migrator
  namespace: github-migrator
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: "arn:aws:acm:..."
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: http
spec:
  type: LoadBalancer
  ports:
  - port: 443
    targetPort: 8080
    protocol: TCP
    name: https
  selector:
    app: github-migrator
```

## PostgreSQL Setup

### Option 1: In-Cluster PostgreSQL

Create `postgres.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: github-migrator
type: Opaque
stringData:
  POSTGRES_USER: migrator
  POSTGRES_PASSWORD: your-secure-password
  POSTGRES_DB: migrator
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-data
  namespace: github-migrator
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: github-migrator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        ports:
        - containerPort: 5432
        envFrom:
        - secretRef:
            name: postgres-secret
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "1Gi"
            cpu: "500m"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: postgres-data
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: github-migrator
spec:
  ports:
  - port: 5432
  selector:
    app: postgres
```

Update the migrator deployment:

```yaml
env:
- name: GHMIG_DATABASE_TYPE
  value: "postgresql"
- name: GHMIG_DATABASE_DSN
  value: "host=postgres port=5432 user=migrator password=your-secure-password dbname=migrator sslmode=disable"
```

### Option 2: Managed PostgreSQL

For production, use managed PostgreSQL services:
- AWS RDS for PostgreSQL
- Azure Database for PostgreSQL
- Google Cloud SQL for PostgreSQL

Update the DSN with your managed database endpoint.

## Scaling

### Horizontal Pod Autoscaler

Create `hpa.yaml`:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: github-migrator
  namespace: github-migrator
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: github-migrator
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

```bash
kubectl apply -f hpa.yaml
```

### Manual Scaling

```bash
kubectl scale deployment github-migrator --replicas=5 -n github-migrator
```

## Monitoring

### Prometheus ServiceMonitor

If using Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: github-migrator
  namespace: github-migrator
spec:
  selector:
    matchLabels:
      app: github-migrator
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

### Health Checks

```bash
# Liveness check
kubectl exec -it deployment/github-migrator -n github-migrator -- curl localhost:8080/health

# Check events
kubectl get events -n github-migrator --sort-by='.lastTimestamp'

# Describe pod for issues
kubectl describe pod -l app=github-migrator -n github-migrator
```

### Log Aggregation

Configure your cluster's log aggregation (EFK, Loki, etc.) to collect logs from `/app/logs/migrator.log`.

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl get pods -n github-migrator

# Check pod events
kubectl describe pod <pod-name> -n github-migrator

# Check logs
kubectl logs <pod-name> -n github-migrator
```

**Common Issues**:
- Image pull errors: Check registry credentials
- CrashLoopBackOff: Check environment variables and config
- Pending: Check PVC binding and node resources

### Database Connection Issues

```bash
# Test PostgreSQL connection
kubectl exec -it deployment/github-migrator -n github-migrator -- \
  sh -c 'nc -zv postgres 5432'

# Check secret values
kubectl get secret github-tokens -n github-migrator -o yaml
```

### Memory Issues

```bash
# Check resource usage
kubectl top pods -n github-migrator

# Increase limits if needed
kubectl patch deployment github-migrator -n github-migrator \
  --patch '{"spec":{"template":{"spec":{"containers":[{"name":"migrator","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
```

### Persistent Volume Issues

```bash
# Check PVC status
kubectl get pvc -n github-migrator

# Check PV binding
kubectl describe pvc migrator-data -n github-migrator
```

## Complete Manifest

For convenience, here's a Kustomize-ready structure:

```
kubernetes/
├── kustomization.yaml
├── namespace.yaml
├── configmap.yaml
├── secret.yaml
├── pvc.yaml
├── deployment.yaml
├── service.yaml
├── ingress.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

`kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: github-migrator
resources:
- namespace.yaml
- configmap.yaml
- secret.yaml
- pvc.yaml
- deployment.yaml
- service.yaml
- ingress.yaml
```

Deploy with:

```bash
kubectl apply -k kubernetes/overlays/prod
```

## Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [cert-manager](https://cert-manager.io/docs/)

