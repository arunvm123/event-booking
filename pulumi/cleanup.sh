#!/bin/bash

set -e

echo "🗑️  Event Booking Application - Infrastructure Cleanup"
echo "=================================================="

echo "⚠️  WARNING: This will delete ALL infrastructure resources!"
echo "   - Kubernetes cluster and all workloads"
echo "   - PostgreSQL database (all data will be lost)"
echo "   - Redis cache"
echo "   - VPC and networking"
echo ""

read -p "Are you absolutely sure you want to delete everything? Type 'DELETE' to confirm: " -r
echo
if [[ $REPLY != "DELETE" ]]; then
    echo "❌ Cleanup cancelled."
    exit 0
fi

echo "🔍 Running Pulumi preview for destruction..."
pulumi preview --diff

echo ""
read -p "Proceed with destroying all resources? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Cleanup cancelled."
    exit 0
fi

echo "🗑️  Destroying infrastructure..."
pulumi destroy --yes

echo ""
echo "✅ All resources have been destroyed!"
echo "💰 Remember to check your DigitalOcean billing to ensure everything is cleaned up."
