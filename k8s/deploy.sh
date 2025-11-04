#!/bin/bash

# K8s 部署脚本
set -e

NAMESPACE="monitoring"
APP_NAME="domain-exporter"

echo "🚀 开始部署 $APP_NAME 到 K8s..."

# 检查 kubectl 是否可用
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl 未找到，请先安装 kubectl"
    exit 1
fi

# 检查集群连接
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ 无法连接到 K8s 集群，请检查 kubeconfig"
    exit 1
fi

echo "✅ K8s 集群连接正常"

# 创建命名空间（如果不存在）
echo "📦 检查命名空间 $NAMESPACE..."
if ! kubectl get namespace $NAMESPACE &> /dev/null; then
    echo "创建命名空间 $NAMESPACE..."
    kubectl create namespace $NAMESPACE
else
    echo "命名空间 $NAMESPACE 已存在"
fi

# 应用配置
echo "🔧 应用 K8s 配置..."
kubectl apply -f deployment.yaml

# 等待部署完成
echo "⏳ 等待部署完成..."
kubectl rollout status deployment/$APP_NAME -n $NAMESPACE --timeout=300s

# 检查 Pod 状态
echo "📊 检查 Pod 状态..."
kubectl get pods -n $NAMESPACE -l app=$APP_NAME

# 检查服务状态
echo "🌐 检查服务状态..."
kubectl get svc -n $NAMESPACE -l app=$APP_NAME

# 显示日志
echo "📋 显示最近的日志..."
kubectl logs -n $NAMESPACE -l app=$APP_NAME --tail=20

# 应用 ServiceMonitor（如果存在 Prometheus Operator）
if kubectl get crd servicemonitors.monitoring.coreos.com &> /dev/null; then
    echo "🔍 应用 ServiceMonitor..."
    kubectl apply -f servicemonitor.yaml
    echo "✅ ServiceMonitor 已应用"
else
    echo "⚠️  未检测到 Prometheus Operator，跳过 ServiceMonitor"
fi

echo "🎉 部署完成！"
echo ""
echo "📝 有用的命令："
echo "  查看日志: kubectl logs -n $NAMESPACE -l app=$APP_NAME -f"
echo "  查看状态: kubectl get pods -n $NAMESPACE -l app=$APP_NAME"
echo "  端口转发: kubectl port-forward -n $NAMESPACE svc/$APP_NAME 8080:8080"
echo "  删除应用: kubectl delete -f deployment.yaml"