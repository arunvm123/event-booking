# Event Booking Application - Kubernetes Deployment Guide

This directory contains Kubernetes manifests for deploying the event booking application to your DigitalOcean Kubernetes cluster.

## Prerequisites

1. **Infrastructure Deployed**: Complete the Pulumi infrastructure deployment first
2. **Docker Images**: Build and push your Docker images to a container registry
3. **Cluster Access**: Configure kubectl to access your DOKS cluster
4. **Domain**: A domain name for external access (optional)

## Quick Start

### 1. Deploy Infrastructure
```bash
# Deploy the infrastructure first
cd ../pulumi
./deploy.sh
```

### 2. Configure Cluster Access
```bash
# Get kubeconfig from Pulumi
cd ../pulumi
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
export KUBECONFIG=./kubeconfig.yaml

# Verify cluster access
kubectl get nodes
kubectl get namespace event-booking
```

### 3. Build and Push Docker Images

#### Option A: Use the automated script (recommended)
```bash
# Set your registry (optional, defaults to ghcr.io/your-username)
export DOCKER_REGISTRY=ghcr.io/your-username
export IMAGE_TAG=latest

# Run the build and push script
./build-and-push.sh
```

#### Option B: Manual build and push
```bash
# Login to your registry first
docker login ghcr.io  # for GitHub Container Registry
# or
docker login  # for Docker Hub

# Build all service images (this only builds locally)
docker build -t ghcr.io/your-username/user-service:latest -f user-service/Dockerfile .
docker build -t ghcr.io/your-username/event-service:latest -f event-service/Dockerfile .
docker build -t ghcr.io/your-username/booking-service-api:latest -f booking-service/Dockerfile.api .
docker build -t ghcr.io/your-username/booking-service-worker:latest -f booking-service/Dockerfile.worker .
docker build -t ghcr.io/your-username/notification-service-api:latest -f notification-service/Dockerfile.api .
docker build -t ghcr.io/your-username/notification-service-worker:latest -f notification-service/Dockerfile.worker .

# Push images to registry (this uploads to the registry)
docker push ghcr.io/your-username/user-service:latest
docker push ghcr.io/your-username/event-service:latest
docker push ghcr.io/your-username/booking-service-api:latest
docker push ghcr.io/your-username/booking-service-worker:latest
docker push ghcr.io/your-username/notification-service-api:latest
docker push ghcr.io/your-username/notification-service-worker:latest
```

### 4. Update Image References
Before deploying, update the image references in the YAML files:

#### On Linux:
```bash
cd k8s
sed -i 's/your-registry/registry.digitalocean.com\\/event-booking-registry/g' *.yaml
```

#### On macOS:
```bash
cd k8s
sed -i '' 's/your-registry/registry.digitalocean.com\\/event-booking-registry/g' *.yaml
```

#### Alternative (works on both Linux and macOS):
```bash
cd k8s
for file in *.yaml; do
  sed 's/your-registry/registry.digitalocean.com\/event-booking-registry/g' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
done
```

### 5. Deploy Services
```bash
# Deploy all services
kubectl apply -f k8s/user-service.yaml
kubectl apply -f k8s/event-service.yaml
kubectl apply -f k8s/booking-service.yaml
kubectl apply -f k8s/notification-service.yaml
```

### 6. Verify Deployment
```bash
# Check all pods are running
kubectl get pods -n event-booking

# Check services
kubectl get services -n event-booking

# Check configuration
kubectl get configmap event-booking-config -n event-booking -o yaml
kubectl get secret event-booking-secret -n event-booking -o yaml
```

## Detailed Deployment Steps

### Step 1: Infrastructure Verification

Ensure your infrastructure is properly deployed:
```bash
# Check managed services are accessible
pulumi stack output databaseHost
pulumi stack output redisHost
pulumi stack output kafkaHost

# Verify ConfigMap and Secret exist
kubectl get configmap -n event-booking
kubectl get secret -n event-booking
```

### Step 2: Database Migration

Run database migrations before deploying services:
```bash
# Create a migration job (example)
kubectl run migration-job --image=your-registry/user-service:latest \
  --rm -it --restart=Never \
  --namespace=event-booking \
  --env="DB_HOST=$(kubectl get configmap event-booking-config -n event-booking -o jsonpath='{.data.DB_HOST}')" \
  --env="DB_PORT=$(kubectl get configmap event-booking-config -n event-booking -o jsonpath='{.data.DB_PORT}')" \
  --env="DB_NAME=$(kubectl get configmap event-booking-config -n event-booking -o jsonpath='{.data.DB_NAME}')" \
  --env="DB_USER=$(kubectl get configmap event-booking-config -n event-booking -o jsonpath='{.data.DB_USER}')" \
  --env="DB_PASSWORD=$(kubectl get secret event-booking-secret -n event-booking -o jsonpath='{.data.DB_PASSWORD}' | base64 -d)" \
  -- /bin/sh -c "go run migration/main.go"
```

### Step 3: Service-by-Service Deployment

Deploy services in dependency order:

1. **User Service** (no dependencies)
```bash
kubectl apply -f k8s/user-service.yaml
kubectl rollout status deployment/user-service -n event-booking
```

2. **Event Service** (depends on User Service for auth)
```bash
kubectl apply -f k8s/event-service.yaml
kubectl rollout status deployment/event-service -n event-booking
```

3. **Booking Service** (depends on Event Service)
```bash
kubectl apply -f k8s/booking-service.yaml
kubectl rollout status deployment/booking-service-api -n event-booking
kubectl rollout status deployment/booking-service-worker -n event-booking
```

4. **Notification Service** (consumes Kafka events)
```bash
kubectl apply -f k8s/notification-service.yaml
kubectl rollout status deployment/notification-service-api -n event-booking
kubectl rollout status deployment/notification-service-worker -n event-booking
```

### Step 4: External Access Setup

#### Option A: LoadBalancer Service (Simple)
```bash
# Expose services via LoadBalancer
kubectl patch service user-service -n event-booking -p '{"spec":{"type":"LoadBalancer"}}'
kubectl patch service event-service -n event-booking -p '{"spec":{"type":"LoadBalancer"}}'
kubectl patch service booking-service -n event-booking -p '{"spec":{"type":"LoadBalancer"}}'

# Get external IPs
kubectl get services -n event-booking
```

#### Option B: Ingress Controller (Recommended)
```bash
# Install NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/do/deploy.yaml

# Wait for controller to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

# Update ingress.yaml with your domain
sed -i 's/api.your-domain.com/api.yourdomain.com/g' k8s/ingress.yaml

# Deploy ingress
kubectl apply -f k8s/ingress.yaml

# Get LoadBalancer IP
kubectl get service ingress-nginx-controller -n ingress-nginx
```

## Configuration Management

### Environment Variables

All services are configured through ConfigMaps and Secrets:

- **ConfigMap** (`event-booking-config`): Non-sensitive configuration
- **Secret** (`event-booking-secret`): Sensitive data (passwords, tokens)

### Updating Configuration

```bash
# Update ConfigMap
kubectl patch configmap event-booking-config -n event-booking --patch '{"data":{"NEW_KEY":"new-value"}}'

# Update Secret
kubectl patch secret event-booking-secret -n event-booking --patch '{"stringData":{"NEW_SECRET":"secret-value"}}'

# Restart deployments to pick up changes
kubectl rollout restart deployment/user-service -n event-booking
kubectl rollout restart deployment/event-service -n event-booking
kubectl rollout restart deployment/booking-service-api -n event-booking
kubectl rollout restart deployment/booking-service-worker -n event-booking
kubectl rollout restart deployment/notification-service-api -n event-booking
kubectl rollout restart deployment/notification-service-worker -n event-booking
```

## Scaling

### Horizontal Pod Autoscaling

```bash
# Enable metrics server (if not already enabled)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Create HPA for services
kubectl autoscale deployment user-service --cpu-percent=70 --min=2 --max=10 -n event-booking
kubectl autoscale deployment event-service --cpu-percent=70 --min=2 --max=10 -n event-booking
kubectl autoscale deployment booking-service-api --cpu-percent=70 --min=2 --max=10 -n event-booking
kubectl autoscale deployment booking-service-worker --cpu-percent=70 --min=2 --max=5 -n event-booking
kubectl autoscale deployment notification-service-api --cpu-percent=70 --min=2 --max=5 -n event-booking
kubectl autoscale deployment notification-service-worker --cpu-percent=70 --min=2 --max=5 -n event-booking

# Check HPA status
kubectl get hpa -n event-booking
```

### Manual Scaling

```bash
# Scale specific services
kubectl scale deployment user-service --replicas=5 -n event-booking
kubectl scale deployment event-service --replicas=5 -n event-booking
kubectl scale deployment booking-service-api --replicas=5 -n event-booking
```

## Monitoring and Troubleshooting

### Check Pod Status
```bash
# Get all pods
kubectl get pods -n event-booking

# Describe specific pod
kubectl describe pod <pod-name> -n event-booking

# View logs
kubectl logs <pod-name> -n event-booking -f

# Execute into pod
kubectl exec -it <pod-name> -n event-booking -- /bin/sh
```

### Service Connectivity Testing
```bash
# Test internal service connectivity
kubectl run test-pod --image=alpine/curl --rm -it --restart=Never -n event-booking -- /bin/sh

# Inside the pod:
curl http://user-service/health
curl http://event-service/health
curl http://booking-service/health
curl http://notification-service/health
```

### Database Connectivity
```bash
# Test database connection
kubectl run pg-test --image=postgres:15 --rm -it --restart=Never -n event-booking \
  --env="PGPASSWORD=$(kubectl get secret event-booking-secret -n event-booking -o jsonpath='{.data.DB_PASSWORD}' | base64 -d)" \
  -- psql -h $(kubectl get configmap event-booking-config -n event-booking -o jsonpath='{.data.DB_HOST}') \
         -U $(kubectl get configmap event-booking-config -n event-booking -o jsonpath='{.data.DB_USER}') \
         -d $(kubectl get configmap event-booking-config -n event-booking -o jsonpath='{.data.DB_NAME}')
```

## SSL/TLS Setup

### Install Cert-Manager
```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create Let's Encrypt issuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@domain.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

## CI/CD Integration

### GitHub Actions Example
```yaml
# .github/workflows/deploy.yml
name: Deploy to Kubernetes
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Build and push images
      run: |
        docker build -t ghcr.io/${{ github.repository }}/user-service:${{ github.sha }} -f user-service/Dockerfile .
        docker push ghcr.io/${{ github.repository }}/user-service:${{ github.sha }}
    - name: Deploy to Kubernetes
      run: |
        sed -i 's/latest/${{ github.sha }}/g' k8s/*.yaml
        kubectl apply -f k8s/
```

## Security Best Practices

1. **Use specific image tags** (not `latest`)
2. **Implement Pod Security Standards**
3. **Use Network Policies** for inter-pod communication
4. **Regular security updates** for base images
5. **Secrets management** with external secret managers
6. **RBAC** for service accounts

## Backup and Disaster Recovery

### Database Backups
DigitalOcean managed databases include automatic backups, but you can also:
```bash
# Manual backup
kubectl run backup-job --image=postgres:15 --rm -it --restart=Never -n event-booking \
  --env="PGPASSWORD=..." \
  -- pg_dump -h <db-host> -U <db-user> -d <db-name> > backup.sql
```

### Configuration Backups
```bash
# Backup all configurations
kubectl get configmap,secret -n event-booking -o yaml > event-booking-config-backup.yaml
```

This guide provides a comprehensive approach to deploying your event booking application to Kubernetes with proper configuration, scaling, monitoring, and security considerations.
