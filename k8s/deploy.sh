#!/bin/bash

set -e

echo "🚀 Event Booking Application - Service Deployment"
echo "=============================================="

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed or not in PATH"
    exit 1
fi

# Check if we can access the cluster
if ! kubectl get nodes &> /dev/null; then
    echo "❌ Cannot access Kubernetes cluster"
    echo "   Make sure kubeconfig is set up correctly:"
    echo "   export KUBECONFIG=../pulumi/kubeconfig.yaml"
    exit 1
fi

# Check if event-booking namespace exists
if ! kubectl get namespace event-booking &> /dev/null; then
    echo "❌ event-booking namespace not found"
    echo "   Make sure infrastructure is deployed first:"
    echo "   cd ../pulumi && ./deploy.sh"
    exit 1
fi

# Function to wait for deployment
wait_for_deployment() {
    local deployment=$1
    local namespace=$2
    echo "⏳ Waiting for deployment $deployment to be ready..."
    kubectl rollout status deployment/$deployment -n $namespace --timeout=300s
    if [ $? -eq 0 ]; then
        echo "✅ Deployment $deployment is ready"
    else
        echo "❌ Deployment $deployment failed to become ready"
        return 1
    fi
}

# Check if images need to be updated
echo "📋 Current image references:"
grep "image:" *.yaml | head -5

echo ""
read -p "Have you updated the image references with your registry? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Please update image references first:"
    echo "   sed -i 's/your-registry/ghcr.io\/your-username/g' k8s/*.yaml"
    exit 0
fi

echo "🔍 Verifying infrastructure components..."

# Check ConfigMap and Secret
if kubectl get configmap event-booking-config -n event-booking &> /dev/null; then
    echo "✅ ConfigMap found"
else
    echo "❌ ConfigMap not found"
    exit 1
fi

if kubectl get secret event-booking-secret -n event-booking &> /dev/null; then
    echo "✅ Secret found"
else
    echo "❌ Secret not found"
    exit 1
fi

echo ""
echo "🚀 Starting service deployment..."

# Deploy services in dependency order
echo ""
echo "1️⃣ Deploying User Service..."
kubectl apply -f user-service.yaml
wait_for_deployment "user-service" "event-booking"

echo ""
echo "2️⃣ Deploying Event Service..."
kubectl apply -f event-service.yaml
wait_for_deployment "event-service" "event-booking"

echo ""
echo "3️⃣ Deploying Booking Service..."
kubectl apply -f booking-service.yaml
wait_for_deployment "booking-service-api" "event-booking"
wait_for_deployment "booking-service-worker" "event-booking"

echo ""
echo "4️⃣ Deploying Notification Service..."
kubectl apply -f notification-service.yaml
wait_for_deployment "notification-service-api" "event-booking"
wait_for_deployment "notification-service-worker" "event-booking"

echo ""
echo "✅ All services deployed successfully!"

echo ""
echo "📊 Deployment Summary:"
kubectl get pods -n event-booking
echo ""
kubectl get services -n event-booking

echo ""
echo "🔗 Service URLs (internal):"
echo "   User Service: http://user-service.event-booking.svc.cluster.local"
echo "   Event Service: http://event-service.event-booking.svc.cluster.local"
echo "   Booking Service: http://booking-service.event-booking.svc.cluster.local"
echo "   Notification Service: http://notification-service.event-booking.svc.cluster.local"

echo ""
echo "🌐 For external access, you can:"
echo "1. Use LoadBalancer services:"
echo "   kubectl patch service user-service -n event-booking -p '{\"spec\":{\"type\":\"LoadBalancer\"}}'"
echo ""
echo "2. Or deploy ingress controller:"
echo "   kubectl apply -f ingress.yaml"
echo ""

echo "📚 For more detailed instructions, see README.md"
echo ""
echo "🎉 Deployment completed successfully!"
