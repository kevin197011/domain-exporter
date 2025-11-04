#!/bin/bash

# Nacos 连接检查脚本
set -e

echo "🔍 Nacos 连接检查工具"
echo "====================="

# 读取环境变量
NACOS_URL=${NACOS_URL:-"http://localhost:8848"}
NACOS_USERNAME=${NACOS_USERNAME:-"nacos"}
NACOS_PASSWORD=${NACOS_PASSWORD:-"nacos"}
NACOS_NAMESPACE_ID=${NACOS_NAMESPACE_ID:-"public"}
NACOS_DATA_ID=${NACOS_DATA_ID:-"domain-exporter"}
NACOS_GROUP=${NACOS_GROUP:-"DEFAULT_GROUP"}

echo "📋 配置信息:"
echo "  Nacos URL: $NACOS_URL"
echo "  用户名: $NACOS_USERNAME"
echo "  命名空间: $NACOS_NAMESPACE_ID"
echo "  Data ID: $NACOS_DATA_ID"
echo "  Group: $NACOS_GROUP"
echo ""

# 检查网络连通性
echo "🌐 检查网络连通性..."
if curl -s --connect-timeout 10 "$NACOS_URL" > /dev/null; then
    echo "✅ 网络连通性正常"
else
    echo "❌ 网络连通性失败"
    exit 1
fi

# 获取访问令牌
echo "🔐 获取访问令牌..."
TOKEN_RESPONSE=$(curl -s -X POST "$NACOS_URL/nacos/v1/auth/login" \
    -d "username=$NACOS_USERNAME&password=$NACOS_PASSWORD" \
    --connect-timeout 10)

if echo "$TOKEN_RESPONSE" | grep -q "accessToken"; then
    ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)
    echo "✅ 访问令牌获取成功"
else
    echo "❌ 访问令牌获取失败"
    echo "响应: $TOKEN_RESPONSE"
    exit 1
fi

# 检查配置是否存在
echo "📄 检查配置文件..."
CONFIG_RESPONSE=$(curl -s "$NACOS_URL/nacos/v1/cs/configs" \
    -G \
    -d "dataId=$NACOS_DATA_ID" \
    -d "group=$NACOS_GROUP" \
    -d "tenant=$NACOS_NAMESPACE_ID" \
    -d "accessToken=$ACCESS_TOKEN" \
    --connect-timeout 10)

if [ -n "$CONFIG_RESPONSE" ] && [ "$CONFIG_RESPONSE" != "config data not exist" ]; then
    echo "✅ 配置文件存在"
    echo "📋 配置内容:"
    echo "$CONFIG_RESPONSE"
else
    echo "❌ 配置文件不存在或为空"
    echo "响应: $CONFIG_RESPONSE"
    echo ""
    echo "💡 请在 Nacos 控制台创建配置:"
    echo "  1. 访问: $NACOS_URL/nacos"
    echo "  2. 登录用户名: $NACOS_USERNAME"
    echo "  3. 切换到命名空间: $NACOS_NAMESPACE_ID"
    echo "  4. 创建配置: Data ID=$NACOS_DATA_ID, Group=$NACOS_GROUP"
    exit 1
fi

echo ""
echo "🎉 所有检查通过！Nacos 配置正常。"