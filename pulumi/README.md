# Event Booking Application - Infrastructure as Code

This directory contains Pulumi infrastructure code written in Go to deploy the event booking application on DigitalOcean with managed services.

## Architecture Overview

The infrastructure includes:

### Managed Services
- **DigitalOcean Kubernetes (DOKS)** - Managed Kubernetes cluster in India region (blr1)
- **DigitalOcean Managed PostgreSQL** - Database for persistent storage
- **DigitalOcean Managed Redis** - Cache and session storage
- **DigitalOcean Managed Kafka** - Message broker for event streaming

### Network
- **VPC** - Private network (10.10.0.0/16) for secure communication
- **Private networking** - All managed services are connected via private network

### Application Services
The following microservices will be deployed (to be implemented):
- User Service - User management and authentication
- Event Service - Event creation and management
- Booking Service - Booking processing with Kafka integration
- Notification Service - Email/SMS notifications

## Prerequisites

1. **DigitalOcean Account** and API token
2. **Pulumi CLI** installed
3. **Go 1.21+** installed
4. **kubectl** for Kubernetes access

## Setup Instructions

### 1. Install Pulumi CLI
```bash
curl -fsSL https://get.pulumi.com | sh
```

### 2. Configure DigitalOcean Token
```bash
# Set your DigitalOcean API token
pulumi config set --secret digitalocean:token YOUR_DO_TOKEN
```

### 3. Configure Project Settings
```bash
# Set the region (default: blr1 - Bangalore, India)
pulumi config set region blr1

# Set node configuration
pulumi config set nodeSize s-2vcpu-4gb
pulumi config set nodeCount 3

# Set environment
pulumi config set environment production
```

### 4. Deploy Infrastructure
```bash
# Initialize Pulumi stack
pulumi stack init production

# Preview the deployment
pulumi preview

# Deploy the infrastructure
pulumi up
```

## What Gets Deployed

### Infrastructure Resources
1. **VPC** - `event-booking-vpc` with IP range 10.10.0.0/16
2. **Kubernetes Cluster** - `event-booking-cluster` with 3 nodes
3. **PostgreSQL Database** - `event-booking-postgres` (v15, 1vCPU/1GB)
4. **Redis Cluster** - `event-booking-redis` (v7, 1vCPU/1GB)
5. **Kafka Cluster** - `event-booking-kafka` (v3.5, 3x 2vCPU/2GB)

### Kubernetes Resources
1. **Namespace** - `event-booking`
2. **ConfigMap** - Database and service configuration
3. **Secret** - Sensitive credentials (including Kafka authentication)

## Accessing the Cluster

After deployment, you can access the Kubernetes cluster:

```bash
# Get the kubeconfig
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml

# Set KUBECONFIG
export KUBECONFIG=./kubeconfig.yaml

# Verify access
kubectl get nodes
kubectl get pods -n event-booking
```

## Database Access

Connect to the PostgreSQL database:

```bash
# Get database connection details
pulumi stack output databaseHost
pulumi stack output databasePort

# The database credentials are stored in Kubernetes secrets
kubectl get secret event-booking-secret -n event-booking -o yaml
```

## Cost Considerations

### Estimated Monthly Costs (USD)
- **DOKS Cluster**: ~$36/month (3 x s-2vcpu-4gb nodes)
- **PostgreSQL**: ~$15/month (db-s-1vcpu-1gb)
- **Redis**: ~$15/month (db-s-1vcpu-1gb)
- **Kafka**: ~$90/month (3 x db-s-2vcpu-2gb nodes)
- **VPC**: Free
- **Load Balancer**: ~$12/month (if using LoadBalancer services)

**Total**: ~$168/month

### Cost Optimization Tips
1. Use smaller node sizes for development
2. Reduce node count for non-production environments
3. Use development-tier databases for testing
4. Consider spot instances where available

## Configuration Reference

### Available Configuration Options

```yaml
# Region configuration
region: blr1  # Bangalore, India

# Kubernetes cluster configuration
nodeSize: s-2vcpu-4gb    # Node size
nodeCount: 3             # Number of nodes

# Environment
environment: production  # Environment tag
```

### Supported Regions
- `blr1` - Bangalore, India (default)
- `sgp1` - Singapore
- `fra1` - Frankfurt, Germany

### Node Sizes
- `s-1vcpu-1gb` - 1 vCPU, 1GB RAM
- `s-2vcpu-4gb` - 2 vCPU, 4GB RAM (recommended)
- `s-4vcpu-8gb` - 4 vCPU, 8GB RAM

## Troubleshooting

### Common Issues

1. **Deployment Fails**
   ```bash
   # Check Pulumi logs
   pulumi logs --follow
   
   # Verify DigitalOcean token
   pulumi config get digitalocean:token
   ```

2. **Kubernetes Access Issues**
   ```bash
   # Refresh kubeconfig
   pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
   export KUBECONFIG=./kubeconfig.yaml
   ```

3. **Kafka Connection Issues**
   ```bash
   # Check Kafka connection details
   pulumi stack output kafkaHost
   pulumi stack output kafkaPort
   
   # Verify Kafka credentials in Kubernetes secret
   kubectl get secret event-booking-secret -n event-booking -o yaml
   ```

## Next Steps

1. **Application Deployment** - Deploy the microservices using the ConfigMap and Secret created
2. **SSL/TLS Setup** - Configure ingress with SSL certificates
3. **Monitoring** - Add Prometheus and Grafana for monitoring
4. **Backup Strategy** - Set up database backups
5. **CI/CD Pipeline** - Automate deployments

## Security Notes

- All managed services are deployed in private network
- Database credentials are stored as Kubernetes secrets
- SSL is required for database connections
- Update JWT secret before production use

## Support

For issues related to:
- **Infrastructure**: Check Pulumi and DigitalOcean documentation
- **Application**: Refer to the main repository README
- **Kubernetes**: Use kubectl for debugging pod issues
