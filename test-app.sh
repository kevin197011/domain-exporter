#!/bin/bash

# Simple test script for domain-exporter

echo "🧪 Testing Domain Exporter Application"
echo "======================================"

# Build the application
echo "📦 Building application..."
if ! go build -o domain-exporter .; then
    echo "❌ Build failed"
    exit 1
fi
echo "✅ Build successful"

# Start the application in background
echo "🚀 Starting application..."
./domain-exporter -config=config.yaml &
APP_PID=$!

# Wait for application to start
echo "⏳ Waiting for application to start..."
sleep 5

# Test health endpoint
echo "🔍 Testing health endpoint..."
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed"
    kill $APP_PID 2>/dev/null
    exit 1
fi

# Test status page
echo "🔍 Testing status page..."
if curl -f http://localhost:8080/ > /dev/null 2>&1; then
    echo "✅ Status page accessible"
else
    echo "❌ Status page failed"
    kill $APP_PID 2>/dev/null
    exit 1
fi

# Test metrics endpoint
echo "🔍 Testing metrics endpoint..."
if curl -f http://localhost:8080/metrics > /dev/null 2>&1; then
    echo "✅ Metrics endpoint accessible"
else
    echo "❌ Metrics endpoint failed"
    kill $APP_PID 2>/dev/null
    exit 1
fi

# Stop the application
echo "🛑 Stopping application..."
kill $APP_PID 2>/dev/null
wait $APP_PID 2>/dev/null

echo ""
echo "🎉 All tests passed!"
echo "✅ Application is working correctly with local configuration"
echo ""
echo "🔗 To run the application:"
echo "   make run"
echo "   or"
echo "   go run . -config=config.yaml"
echo ""
echo "🌐 Access points:"
echo "   Status page: http://localhost:8080"
echo "   Metrics: http://localhost:8080/metrics"
echo "   Health check: http://localhost:8080/health"