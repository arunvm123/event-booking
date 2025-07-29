# Pulumi Infrastructure Setup Guide

This guide will walk you through setting up Pulumi and deploying your event booking infrastructure on DigitalOcean.

## Prerequisites

1. **DigitalOcean Account** with billing enabled
2. **Go 1.21+** installed
3. **kubectl** installed
4. **Docker** installed and configured

## Step 1: Install Pulumi CLI

### macOS
```bash
brew install pulumi
```

### Linux
```bash
curl -fsSL https://get.pulumi.com | sh
export PATH=$PATH:$HOME/.pulumi/bin
```

### Windows
```bash
# Using Chocolatey
choco install pulumi

# Or download from https://www.pulumi.com/docs/install/
```

### Verify Installation
```bash
pulumi version
```

## Step 2: Install Additional Tools

```bash
# Install DigitalOcean CLI (optional but helpful)
# macOS
brew install doctl

# Linux
snap install doctl

# Verify
doctl version
```

## Step 3: Configure DigitalOcean Access

### Create Personal Access Token
1. Go to [DigitalOcean API Tokens](https://cloud.digitalocean.com/account/api/tokens)
2. Click "Generate New Token"
3. Name: `pulumi-event-booking`
4. Scopes: **Select all permissions** (Full Access)
5. Expiration: Choose appropriate duration
6. Copy the token and save it securely

### Set Environment Variable
```bash
# Set your DigitalOcean token
export DIGITALOCEAN_TOKEN=your_token_here

# Add to your shell profile for persistence
echo 'export DIGITALOCEAN_TOKEN=your_token_here' >> ~/.bashrc  # or ~/.zshrc
source ~/.bashrc  # or ~/.zshrc
```

## Step 4: Initialize Pulumi Project

```bash
# Navigate to pulumi directory
cd pulumi

# Login to Pulumi (choose one)
pulumi login  # Use Pulumi Cloud (free for individuals)
# OR
pulumi login --local  # Use local file-based state

# Install Go dependencies
go mod tidy

# Create or select production stack
pulumi stack init production
# OR if stack exists
pulumi stack select production
```

## Step 5: Deploy Infrastructure

### Option A: Use Automated Deploy Script (Recommended)
```bash
# The deploy script will handle all configuration and deployment
./deploy.sh
```

### Option B: Manual Configuration and Deployment
```bash
# Set your DigitalOcean token (will be encrypted)
pulumi config set digitalocean:token $DIGITALOCEAN_TOKEN --secret

# Set infrastructure configuration (optional - already configured)
pulumi config set region blr1
pulumi config set nodeSize s-2vcpu-4gb
pulumi config set nodeCount 3
pulumi config set environment production

# Preview deployment
pulumi preview

# Deploy infrastructure
pulumi up
```

## Step 6: Verify Deployment

### Check Stack Outputs
```bash
# View all outputs
pulumi stack output

# Check specific resources
pulumi stack output clusterName      # event-booking-cluster
pulumi stack output databaseHost     # PostgreSQL host
pulumi stack output redisHost        # Redis host  
pulumi stack output kafkaHost        # Kafka host
pulumi stack output vpcId           # VPC ID
```

### Test Cluster Access
```bash
# Get kubeconfig
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
export KUBECONFIG=./kubeconfig.yaml

# Test cluster connectivity
kubectl get nodes
kubectl get namespace event-booking
kubectl get configmap -n event-booking
kubectl get secret -n event-booking
```

### Verify ConfigMap and Secret
```bash
# Check ConfigMap with connection details
kubectl get configmap event-booking-config -n event-booking -o yaml

# Check Secret (passwords are base64 encoded)
kubectl get secret event-booking-secret -n event-booking -o yaml
```

## Step 7: Deploy Container Registry

```bash
# Create DigitalOcean Container Registry
doctl registry create event-booking-registry --region blr1

# Login to registry
doctl registry login
```

## Step 8: Deploy Applications

Now that infrastructure is ready, deploy your applications:

```bash
# Go back to project root
cd ..

# Build and push Docker images
export DOCKER_REGISTRY=registry.digitalocean.com/event-booking-registry
./build-and-push.sh

# Deploy to Kubernetes
cd k8s
./deploy.sh

# Verify deployment
kubectl get pods -n event-booking
kubectl get services -n event-booking
```

## Infrastructure Components Created

Your Pulumi deployment creates:

### 1. **VPC Network**
- **CIDR**: 10.10.0.0/16
- **Region**: blr1 (Bangalore, India)
- **Private networking** for all resources

### 2. **DigitalOcean Kubernetes Service (DOKS)**
- **Name**: event-booking-cluster
- **Region**: blr1 (Bangalore, India)
- **Node Pool**: 3 nodes, s-2vcpu-4gb droplets (4GB RAM, 2 vCPUs each)
- **Version**: 1.28.2-do.0
- **Auto-upgrade**: Enabled
- **VPC**: Connected to private network

### 3. **Managed PostgreSQL Database**
- **Name**: event-booking-postgres
- **Size**: db-s-1vcpu-1gb (1GB RAM, 1 vCPU)
- **Version**: PostgreSQL 15
- **Region**: blr1
- **Storage**: 10GB
- **Automatic backups**: Enabled
- **SSL**: Required

### 4. **Managed Redis Cache**
- **Name**: event-booking-redis
- **Size**: db-s-1vcpu-1gb (1GB RAM, 1 vCPU)
- **Version**: Redis 7
- **Region**: blr1
- **Memory**: 1GB
- **Eviction policy**: allkeys-lru

### 5. **Managed Kafka Cluster**
- **Name**: event-booking-kafka
- **Size**: db-s-2vcpu-2gb per node (2GB RAM, 2 vCPUs)
- **Node Count**: 3 (High Availability)
- **Version**: Kafka 3.5
- **Region**: blr1
- **Storage**: 61GB per node
- **Replication**: 3x

### 6. **Kubernetes Resources**
- **Namespace**: event-booking
- **ConfigMap**: event-booking-config (DB, Redis, Kafka connection details)
- **Secret**: event-booking-secret (passwords and sensitive data)

## Cost Estimation

### Monthly Costs (Approximate)
- **DOKS Cluster**: 3 × $24 = $72/month (s-2vcpu-4gb nodes)
- **PostgreSQL**: $15/month (db-s-1vcpu-1gb)
- **Redis**: $15/month (db-s-1vcpu-1gb)
- **Kafka**: 3 × $60 = $180/month (db-s-2vcpu-2gb × 3 nodes)
- **Container Registry**: $20/month (basic plan)
- **Load Balancers**: $12/month each (as needed)

**Total Infrastructure**: ~$302/month

### Cost Optimization Tips
1. **Development Environment**: Use smaller instances
   ```bash
   pulumi config set nodeSize s-1vcpu-2gb      # $12/node vs $24/node
   pulumi config set nodeCount 2               # 2 nodes vs 3
   ```
2. **Scale down Kafka** to single node for testing: ~$120/month savings
3. **Use self-hosted Kafka** on K8s to save ~$150/month
4. **Smaller databases** for non-production: basic plans

## Configuration Management

Your stack supports these configuration options:

```bash
# Infrastructure sizing
pulumi config set region blr1                    # DigitalOcean region
pulumi config set nodeSize s-2vcpu-4gb          # Kubernetes node size
pulumi config set nodeCount 3                   # Number of K8s nodes
pulumi config set environment production        # Environment tag

# View current config
pulumi config
```

## Troubleshooting

### Common Issues

#### 1. Authentication Errors
```bash
# Verify token is set
echo $DIGITALOCEAN_TOKEN

# Re-authenticate with doctl
doctl auth init

# Check token permissions
doctl account get
```

#### 2. Resource Limits
```bash
# Check account limits and current usage
doctl account get
doctl compute droplet list
doctl database list
doctl kubernetes cluster list
```

#### 3. Kubernetes Access Issues  
```bash
# Regenerate kubeconfig
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
export KUBECONFIG=./kubeconfig.yaml

# Test connectivity
kubectl cluster-info
kubectl get nodes
```

#### 4. Database Connection Issues
```bash
# Check database cluster status
pulumi stack output databaseHost
doctl databases list

# Test connection from within cluster
kubectl run test-db --image=postgres:15 --rm -it --restart=Never -n event-booking \
  --env="PGPASSWORD=$(kubectl get secret event-booking-secret -n event-booking -o jsonpath='{.data.DB_PASSWORD}' | base64 -d)" \
  -- psql -h $(pulumi stack output databaseHost) -U doadmin -d defaultdb -c "SELECT version();"
```

#### 5. Pulumi State Issues
```bash
# Check current stack
pulumi stack ls

# Refresh state
pulumi refresh

# Import existing resources if needed
pulumi import digitalocean:index/vpc:Vpc event-booking-vpc vpc-id-here
```

## Automated Deploy Script Features

Your `deploy.sh` script includes:

- ✅ **Prerequisites Check**: Verifies Pulumi CLI and DigitalOcean token
- ✅ **Automatic Configuration**: Sets defaults if not configured
- ✅ **Interactive Confirmation**: Preview before deployment
- ✅ **Post-deployment Instructions**: Clear next steps
- ✅ **Output Display**: Shows important connection details

## Connection Details for Applications

Your applications will connect to services using these details from the ConfigMap:

```yaml
# From event-booking-config ConfigMap
DB_HOST: <postgres-host>
DB_PORT: "25060"
DB_NAME: "defaultdb"  
DB_USER: "doadmin"
DB_SSL_MODE: "require"

REDIS_HOST: <redis-host>
REDIS_PORT: "25061"

KAFKA_BROKERS: <kafka-host>:9092

JWT_SECRET: "your-jwt-secret-change-in-production"
ENVIRONMENT: "production"

# From event-booking-secret Secret (base64 encoded)
DB_PASSWORD: <postgres-password>
REDIS_PASSWORD: <redis-password>
KAFKA_PASSWORD: <kafka-password>
```

## Cleanup

To destroy all infrastructure:

```bash
# Use cleanup script
./cleanup.sh

# Or manual cleanup
pulumi destroy

# Confirm deletion when prompted
# This will take 10-15 minutes

# Remove local state (if using local backend)
pulumi stack rm production
```

## Security Best Practices

1. **VPC Isolation**: All resources are in private VPC (10.10.0.0/16)
2. **Database Security**: SSL required, private networking only
3. **Secrets Management**: Sensitive data stored in Kubernetes secrets
4. **Access Control**: Use RBAC for Kubernetes access
5. **Token Security**: Keep DigitalOcean tokens secure and rotate regularly
6. **Network Policies**: Consider implementing K8s network policies

## Complete Deployment Workflow

```bash
# 1. Install prerequisites
brew install pulumi doctl

# 2. Set DigitalOcean token  
export DIGITALOCEAN_TOKEN=your_token_here

# 3. Initialize and deploy infrastructure
cd pulumi
pulumi login
go mod tidy
pulumi stack select production
./deploy.sh

# 4. Set up cluster access
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
export KUBECONFIG=./kubeconfig.yaml

# 5. Create container registry
doctl registry create event-booking-registry --region blr1
doctl registry login

# 6. Deploy applications
cd ..
export DOCKER_REGISTRY=registry.digitalocean.com/event-booking-registry
./build-and-push.sh
cd k8s && ./deploy.sh

# 7. Verify everything is running
kubectl get pods -n event-booking
kubectl get services -n event-booking
```

## Next Steps

After successful deployment:

1. **Test Services**: Verify all microservices can connect to databases
2. **DNS & Ingress**: Set up domain names and SSL certificates
3. **Monitoring**: Deploy Prometheus/Grafana for observability
4. **Logging**: Set up centralized logging (ELK stack or similar)
5. **Backups**: Configure automated backups for critical data
6. **CI/CD**: Set up automated deployments with GitHub Actions
7. **Alerting**: Configure monitoring alerts for critical components
8. **Security**: Implement network policies and pod security standards

## Support Resources

- [DigitalOcean Documentation](https://docs.digitalocean.com/)
- [Pulumi Documentation](https://www.pulumi.com/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [DigitalOcean Status](https://status.digitalocean.com/)

Your event booking infrastructure is now ready for production workloads with managed databases, caching, messaging, and container orchestration!
