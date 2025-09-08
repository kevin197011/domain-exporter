#!/bin/bash

# Test script for Helm deployment

set -e

echo "🧪 Testing Helm Deployment"
echo "=========================="

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    echo "❌ Helm is not installed"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed"
    exit 1
fi

# Validate Helm chart
echo "📋 Validating Helm chart..."
if helm lint ./helm/domain-exporter; then
    echo "✅ Helm chart validation passed"
else
    echo "❌ Helm chart validation failed"
    exit 1
fi

# Dry run to check template rendering
echo "🔍 Testing template rendering..."
if helm template test-release ./helm/domain-exporter > /tmp/helm-output.yaml; then
    echo "✅ Template rendering successful"
    echo "📄 Generated manifests saved to /tmp/helm-output.yaml"
else
    echo "❌ Template rendering failed"
    exit 1
fi

# Check if the rendered template contains correct paths
echo "🔍 Checking configuration paths..."
if grep -q '\\-config=/root/config.yaml' /tmp/helm-output.yaml; then
    echo "✅ Configuration path is correct: -config=/root/config.yaml"
else
    echo "❌ Configuration path is incorrect"
    cat /tmp/helm-output.yaml | grep -A5 -B5 'args:'
    exit 1
fi

if grep -q 'mountPath: /root/config.yaml' /tmp/helm-output.yaml; then
    echo "✅ Mount path is correct: /root/config.yaml"
else
    echo "❌ Mount path is incorrect"
    cat /tmp/helm-output.yaml | grep -A5 -B5 'mountPath:'
    exit 1
fi

# Show key parts of the generated manifest
echo ""
echo "📋 Key configuration from generated manifest:"
echo "=============================================="
echo ""
echo "Container args:"
grep -A2 'args:' /tmp/helm-output.yaml || true
echo ""
echo "Volume mounts:"
grep -A5 'volumeMounts:' /tmp/helm-output.yaml || true
echo ""
echo "Volumes:"
grep -A5 'volumes:' /tmp/helm-output.yaml || true

echo ""
echo "🎉 Helm chart validation completed successfully!"
echo ""
echo "🚀 To deploy to Kubernetes:"
echo "   helm install domain-exporter ./helm/domain-exporter"
echo ""
echo "🔧 To deploy with custom values:"
echo "   helm install domain-exporter ./helm/domain-exporter \\\\"
echo "     --set config.domains='{yourdomain.com,api.yourdomain.com}'"
echo ""
echo "📊 To check deployment status:"
echo "   kubectl get pods -l app.kubernetes.io/name=domain-exporter"
echo "   kubectl logs -l app.kubernetes.io/name=domain-exporter"