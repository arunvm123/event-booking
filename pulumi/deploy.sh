#!/bin/bash

set -e

echo "🚀 Event Booking Application - Infrastructure Deployment"
echo "=================================================="

# Check if Pulumi is installed
if ! command -v pulumi &> /dev/null; then
    echo "❌ Pulumi CLI is not installed. Please install it first:"
    echo "   curl -fsSL https://get.pulumi.com | sh"
    exit 1
fi

# Check if DigitalOcean token is set
if [ -z "$DIGITALOCEAN_TOKEN" ]; then
    echo "❌ DIGITALOCEAN_TOKEN environment variable is not set."
    echo "   Please set your DigitalOcean API token:"
    echo "   export DIGITALOCEAN_TOKEN=your_token_here"
    exit 1
fi

# Set the token in Pulumi config
echo "🔑 Setting DigitalOcean token..."
pulumi config set --secret digitalocean:token "$DIGITALOCEAN_TOKEN"

# Set default configuration if not already set
echo "⚙️  Setting default configuration..."

if ! pulumi config get region &> /dev/null; then
    pulumi config set region blr1
    echo "   Region set to: blr1 (Bangalore, India)"
fi

if ! pulumi config get nodeSize &> /dev/null; then
    pulumi config set nodeSize s-2vcpu-4gb
    echo "   Node size set to: s-2vcpu-4gb"
fi

if ! pulumi config get nodeCount &> /dev/null; then
    pulumi config set nodeCount 3
    echo "   Node count set to: 3"
fi

if ! pulumi config get environment &> /dev/null; then
    pulumi config set environment production
    echo "   Environment set to: production"
fi

echo ""
echo "📋 Current Configuration:"
echo "   Region: $(pulumi config get region)"
echo "   Node Size: $(pulumi config get nodeSize)"
echo "   Node Count: $(pulumi config get nodeCount)"
echo "   Environment: $(pulumi config get environment)"
echo ""

# Ask for confirmation
read -p "Do you want to proceed with the deployment? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Deployment cancelled."
    exit 0
fi

echo "🔍 Running Pulumi preview..."
pulumi preview

echo ""
read -p "Does the preview look correct? Proceed with deployment? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Deployment cancelled."
    exit 0
fi

echo "🚀 Deploying infrastructure..."
pulumi up --yes

echo ""
echo "✅ Deployment completed successfully!"
echo ""
echo "📝 Next Steps:"
echo "1. Get kubeconfig:"
echo "   pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml"
echo "   export KUBECONFIG=./kubeconfig.yaml"
echo ""
echo "2. Verify cluster access:"
echo "   kubectl get nodes"
echo "   kubectl get pods -n event-booking"
echo ""
echo "3. Check deployed resources:"
echo "   kubectl get configmap -n event-booking"
echo "   kubectl get secret -n event-booking"
echo ""
echo "🔗 Important Outputs:"
echo "   Cluster Name: $(pulumi stack output clusterName)"
echo "   Database Host: $(pulumi stack output databaseHost)"
echo "   Redis Host: $(pulumi stack output redisHost)"
echo "   Kafka Host: $(pulumi stack output kafkaHost)"
echo "   VPC ID: $(pulumi stack output vpcId)"
echo ""
echo "📚 For more information, see README.md"
