#!/bin/bash

set -e

echo "🔄 Updating Docker Registry References in Kubernetes Manifests"
echo "============================================================="

# Check if registry URL is provided
if [ -z "$1" ]; then
    echo "❌ Please provide the registry URL as an argument"
    echo "Usage: ./update-registry.sh <registry-url>"
    echo "Example: ./update-registry.sh registry.digitalocean.com/event-booking-registry"
    exit 1
fi

REGISTRY_URL="$1"
echo "📋 Registry URL: $REGISTRY_URL"
echo ""

# Function to update files (works on both Linux and macOS)
update_registry() {
    local file=$1
    local registry=$2
    
    echo "   Updating $file..."
    
    # Create a temporary file
    sed "s|your-registry|$registry|g" "$file" > "$file.tmp"
    
    # Replace original file
    mv "$file.tmp" "$file"
}

# Update all YAML files
echo "🔍 Updating registry references..."
for file in *.yaml; do
    if [ -f "$file" ]; then
        update_registry "$file" "$REGISTRY_URL"
    fi
done

echo ""
echo "✅ All manifests updated successfully!"
echo ""
echo "📋 Updated files:"
grep -l "image:" *.yaml | while read file; do
    echo "   $file:"
    grep "image:" "$file" | head -3 | sed 's/^/      /'
    echo ""
done

echo "🚀 Ready to deploy! Run: ./deploy.sh"
