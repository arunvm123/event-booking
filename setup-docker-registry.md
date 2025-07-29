# DigitalOcean Container Registry Setup Guide

This guide will walk you through setting up DigitalOcean's managed Container Registry to store your Docker images for the event booking application.

## What is DigitalOcean Container Registry?

DigitalOcean Container Registry is a managed private Docker registry service that:
- Integrates seamlessly with DOKS (DigitalOcean Kubernetes Service)
- Provides secure private image storage
- Offers automatic vulnerability scanning
- Has built-in access controls
- Supports both public and private repositories

## Step 1: Create a Container Registry

### Via DigitalOcean Web Console

1. **Log in to DigitalOcean Console**
   - Go to [https://cloud.digitalocean.com](https://cloud.digitalocean.com)
   - Sign in to your account

2. **Navigate to Container Registry**
   - Click on "Container Registry" in the left sidebar
   - Or search for "Container Registry" in the top search bar

3. **Create a New Registry**
   - Click "Create a Container Registry"
   - Choose your registry name (e.g., `event-booking-registry`)
   - Select the region closest to your infrastructure (recommend `blr1` - Bangalore)
   - Choose a plan:
     - **Starter**: $5/month, 500MB storage, 500MB bandwidth
     - **Basic**: $20/month, 5GB storage, 1TB bandwidth
     - **Professional**: $50/month, 100GB storage, 5TB bandwidth

4. **Create the Registry**
   - Click "Create Registry"
   - Wait for the registry to be provisioned (usually takes 1-2 minutes)

### Via DigitalOcean CLI (doctl)

```bash
# Install doctl if not already installed
# macOS
brew install doctl

# Linux
curl -sL https://github.com/digitalocean/doctl/releases/download/v1.104.0/doctl-1.104.0-linux-amd64.tar.gz | tar -xzv
sudo mv doctl /usr/local/bin

# Authenticate with DigitalOcean
doctl auth init

# Create a container registry
doctl registry create event-booking-registry --region blr1
```

## Step 2: Configure Docker Authentication

### Method A: Using doctl (Recommended)

```bash
# Install doctl and authenticate (as shown above)

# Configure Docker to authenticate with your registry
doctl registry login

# This configures Docker to use your DO registry
# The login is valid for 12 hours
```

### Method B: Using Personal Access Token

1. **Create a Personal Access Token**
   - Go to API → Personal Access Tokens in DO Console
   - Click "Generate New Token"
   - Name: `container-registry-access`
   - Scopes: Select "read" and "write" for registry
   - Click "Generate Token"
   - **Copy and save the token securely**

2. **Login to Docker Registry**
   ```bash
   # Login using your token
   echo "YOUR_PERSONAL_ACCESS_TOKEN" | docker login registry.digitalocean.com -u YOUR_DO_EMAIL --password-stdin
   ```

## Step 3: Configure Your Registry URL

Your registry URL will be in the format:
```
registry.digitalocean.com/your-registry-name
```

For example, if your registry name is `event-booking-registry`:
```
registry.digitalocean.com/event-booking-registry
```

## Step 4: Update Your Build Scripts

### Update Environment Variables

```bash
# Set your DigitalOcean registry
export DOCKER_REGISTRY=registry.digitalocean.com/event-booking-registry
export IMAGE_TAG=latest

# Run the build and push script
./build-and-push.sh
```

### Update Kubernetes Manifests

#### On Linux:
```bash
cd k8s
sed -i 's/your-registry/registry.digitalocean.com\/event-booking-registry/g' *.yaml
```

#### On macOS:
```bash
cd k8s
sed -i '' 's/your-registry/registry.digitalocean.com\/event-booking-registry/g' *.yaml
```

#### Alternative (works on both):
```bash
cd k8s
for file in *.yaml; do
  sed 's/your-registry/registry.digitalocean.com\/event-booking-registry/g' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
done
```

## Step 5: Integrate Registry with DOKS

DigitalOcean Kubernetes automatically has access to your container registry in the same account. No additional configuration needed!

### Verify Integration

```bash
# Check if your cluster can pull from the registry
kubectl create secret docker-registry do-registry \
  --docker-server=registry.digitalocean.com \
  --docker-username=<your-do-email> \
  --docker-password=<your-personal-access-token> \
  --namespace=event-booking

# Test pulling an image (after you've pushed one)
kubectl run test-pull --image=registry.digitalocean.com/event-booking-registry/user-service:latest \
  --rm -it --restart=Never --namespace=event-booking -- echo "Registry access successful"
```

## Step 6: Build and Push Your Images

### Using the Automated Script

```bash
# Set your DO registry
export DOCKER_REGISTRY=registry.digitalocean.com/event-booking-registry

# Authenticate Docker
doctl registry login

# Build and push all images
./build-and-push.sh
```

### Manual Process

```bash
# Authenticate
doctl registry login

# Build and push each service
docker build -t registry.digitalocean.com/event-booking-registry/user-service:latest -f user-service/Dockerfile .
docker push registry.digitalocean.com/event-booking-registry/user-service:latest

docker build -t registry.digitalocean.com/event-booking-registry/event-service:latest -f event-service/Dockerfile .
docker push registry.digitalocean.com/event-booking-registry/event-service:latest

docker build -t registry.digitalocean.com/event-booking-registry/booking-service-api:latest -f booking-service/Dockerfile.api .
docker push registry.digitalocean.com/event-booking-registry/booking-service-api:latest

docker build -t registry.digitalocean.com/event-booking-registry/booking-service-worker:latest -f booking-service/Dockerfile.worker .
docker push registry.digitalocean.com/event-booking-registry/booking-service-worker:latest

docker build -t registry.digitalocean.com/event-booking-registry/notification-service-api:latest -f notification-service/Dockerfile.api .
docker push registry.digitalocean.com/event-booking-registry/notification-service-api:latest

docker build -t registry.digitalocean.com/event-booking-registry/notification-service-worker:latest -f notification-service/Dockerfile.worker .
docker push registry.digitalocean.com/event-booking-registry/notification-service-worker:latest
```

## Step 7: Deploy to Kubernetes

After pushing your images:

```bash
# Deploy services
cd k8s
./deploy.sh
```

## Registry Management

### View Your Images

```bash
# List repositories in your registry
doctl registry repository list

# List tags for a specific repository
doctl registry repository list-tags event-booking-registry/user-service
```

### Web Console Management

1. Go to Container Registry in DO Console
2. Click on your registry name
3. Browse repositories and tags
4. View vulnerability scans
5. Manage access and permissions

## Cost Management

### Registry Pricing (as of 2024)

- **Starter**: $5/month
  - 500MB storage
  - 500MB outbound data transfer
  - Good for small projects

- **Basic**: $20/month
  - 5GB storage
  - 1TB outbound data transfer
  - Recommended for production

### Cost Optimization Tips

1. **Use multi-stage builds** to reduce image sizes
2. **Clean up old images** regularly
3. **Use specific tags** instead of always using `latest`
4. **Monitor storage usage** in the DO console

### Cleanup Old Images

```bash
# List all tags for a repository
doctl registry repository list-tags event-booking-registry/user-service

# Delete specific tags
doctl registry repository delete-tag event-booking-registry/user-service old-tag-name

# Run garbage collection to free up storage
doctl registry garbage-collection start event-booking-registry
```

## Security Best Practices

### 1. Use Private Repositories
- Keep your repositories private unless you need them public
- DigitalOcean registries are private by default

### 2. Regular Token Rotation
```bash
# Rotate personal access tokens regularly
# Update tokens in CI/CD systems when rotated
```

### 3. Enable Vulnerability Scanning
- DigitalOcean automatically scans images for vulnerabilities
- Review scan results in the web console
- Update base images regularly

### 4. Use Specific Image Tags
```bash
# Instead of 'latest', use specific versions
docker tag registry.digitalocean.com/event-booking-registry/user-service:latest \
         registry.digitalocean.com/event-booking-registry/user-service:v1.0.0

docker push registry.digitalocean.com/event-booking-registry/user-service:v1.0.0
```

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   ```bash
   # Re-authenticate
   doctl registry login
   
   # Or check if token is expired
   doctl auth list
   ```

2. **Push Denied**
   ```bash
   # Ensure you have write permissions
   # Check if registry name is correct
   # Verify you're authenticated to the right account
   ```

3. **Pull Denied in Kubernetes**
   ```bash
   # Ensure DOKS cluster is in the same account
   # Check if image name and tag are correct
   # Verify image exists in registry
   ```

## Alternative: GitHub Container Registry

If you prefer using GitHub Container Registry (which is free for public repos):

```bash
# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u your-username --password-stdin

# Use in your environment
export DOCKER_REGISTRY=ghcr.io/your-username
./build-and-push.sh
```

## Complete Example Workflow

```bash
# 1. Set up registry (one-time)
doctl registry create event-booking-registry --region blr1

# 2. Authenticate Docker
doctl registry login

# 3. Set environment variables
export DOCKER_REGISTRY=registry.digitalocean.com/event-booking-registry
export IMAGE_TAG=v1.0.0

# 4. Build and push images
./build-and-push.sh

# 5. Update Kubernetes manifests
cd k8s
sed -i 's/your-registry/registry.digitalocean.com\/event-booking-registry/g' *.yaml
sed -i 's/latest/v1.0.0/g' *.yaml

# 6. Deploy to Kubernetes
./deploy.sh
```

This setup provides a robust, production-ready container registry that integrates seamlessly with your DigitalOcean Kubernetes cluster and scales with your application needs.
