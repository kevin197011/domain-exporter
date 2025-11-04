#!/bin/bash

# 部署脚本 - 跳过 HTTPS SSL 证书验证

set -e

NAMESPACE="monitoring"
RELEASE_NAME="domain-exporter"

echo "=== 部署域名监控服务（跳过SSL验证）==="

echo "1. 验证 Helm Chart..."
helm lint ./helm/domain-exporter

echo ""
echo "2. 部署或升级服务（跳过SSL验证）..."
helm upgrade --install $RELEASE_NAME ./helm/domain-exporter \
  -n $NAMESPACE \
  --create-namespace \
  --set nacos.enabled=true \
  --set nacos.url="https://infra-nacos.slileisure.com" \
  --set nacos.namespaceId="devops" \
  --set nacos.username="nacos" \
  --set nacos.password="nacos" \
  --set nacos.skipSSLVerify=true \
  --set config.domains="example.com,test.com" \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.labels.release=prometheus \
  --wait

echo ""
echo "3. 检查部署状态..."
kubectl -n $NAMESPACE get all -l app.kubernetes.io/name=domain-exporter

echo ""
echo "4. 等待 Pod 就绪..."
kubectl -n $NAMESPACE wait --for=condition=ready pod -l app.kubernetes.io/name=domain-exporter --timeout=60s

echo ""
echo "5. 查看日志..."
POD_NAME=$(kubectl -n $NAMESPACE get pods -l app.kubernetes.io/name=domain-exporter -o jsonpath='{.items[0].metadata.name}')
echo "Pod: $POD_NAME"
kubectl -n $NAMESPACE logs $POD_NAME --tail=15

echo ""
echo "=== 部署完成 ==="
echo "⚠️  注意：已跳过SSL证书验证，仅适用于开发/测试环境"
echo "监控端点: kubectl -n $NAMESPACE port-forward $POD_NAME 8080:8080"