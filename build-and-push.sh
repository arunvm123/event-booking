#!/bin/bash

set -e

echo "🐳 Event Booking Application - Build and Push Docker Images"
echo "========================================================="

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running or not accessible"
    exit 1
fi

# Default registry - can be overridden
REGISTRY=${DOCKER_REGISTRY:-"ghcr.io/your-username"}
TAG=${IMAGE_TAG:-"latest"}

echo "📋 Configuration:"
echo "   Registry: $REGISTRY"
echo "   Tag: $TAG"
echo ""

# Function to build and push an image
build_and_push() {
    local service=$1
    local dockerfile=$2
    local image_name="$REGISTRY/$service:$TAG"
    
    echo "🔨 Building $service..."
    docker build -t $image_name -f $dockerfile .
    
    echo "📤 Pushing $service to registry..."
    docker push $image_name
    
    echo "✅ $service built and pushed successfully"
    echo ""
}

# Check if user wants to proceed
echo "This will build and push the following images:"
echo "  - $REGISTRY/user-service:$TAG"
echo "  - $REGISTRY/event-service:$TAG"
echo "  - $REGISTRY/booking-service-api:$TAG"
echo "  - $REGISTRY/booking-service-worker:$TAG"
echo "  - $REGISTRY/notification-service-api:$TAG"
echo ""

read -p "Do you want to continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Build cancelled."
    exit 0
fi

# Login to registry (if needed)
echo "🔐 Docker registry login..."
echo "If prompted, please log in to your Docker registry"
echo "For GitHub Container Registry: docker login ghcr.io"
echo "For Docker Hub: docker login"
echo ""

# Build and push all services
build_and_push "user-service" "user-service/Dockerfile"
build_and_push "event-service" "event-service/Dockerfile"
build_and_push "booking-service-api" "booking-service/Dockerfile.api"
build_and_push "booking-service-worker" "booking-service/Dockerfile.worker"
build_and_push "notification-service-api" "notification-service/Dockerfile.api"

echo "🎉 All images built and pushed successfully!"
echo ""
echo "📝 Next steps:"
echo "1. Update Kubernetes manifests with your registry:"
echo "   cd k8s"
echo "   sed -i 's/your-registry/$REGISTRY/g' *.yaml"
echo ""
echo "2. Deploy to Kubernetes:"
echo "   ./deploy.sh"
echo ""
echo "🔗 Your images are now available at:"
echo "   $REGISTRY/user-service:$TAG"
echo "   $REGISTRY/event-service:$TAG"
echo "   $REGISTRY/booking-service-api:$TAG"
echo "   $REGISTRY/booking-service-worker:$TAG"
echo "   $REGISTRY/notification-service-api:$TAG"
