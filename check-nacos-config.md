# Nacos 配置检查清单

## 🔍 当前问题分析

从日志看到：`namespaceId=devops`，但配置加载失败。

## 📋 检查步骤

### 1. 检查 Pod 环境变量
```bash
kubectl -n monitoring exec domain-exporter-cbdf49596-7w854 -- env | grep NACOS
```

### 2. 检查 Nacos 服务器连通性
```bash
# 进入 Pod 测试网络
kubectl -n monitoring exec -it domain-exporter-cbdf49596-7w854 -- sh

# 在 Pod 内测试连接
nc -zv 192.168.1.11 8848
# 或者
telnet 192.168.1.11 8848
```

### 3. 检查 Nacos 控制台

访问：http://192.168.1.11:8848/nacos

1. **检查命名空间**：
   - 是否存在 `devops` 命名空间？
   - 如果不存在，需要创建

2. **检查配置**：
   - 命名空间：`devops`
   - Data ID：`domain-exporter`
   - Group：`DEFAULT_GROUP`

### 4. 创建 Nacos 配置

如果配置不存在，在 Nacos 控制台创建：

**配置内容**：
```yaml
domains:
  - "example.com"
  - "test.com"
  - "yourdomain.com"

check_interval: 3600
port: 8080
log_level: "info"
timeout: 30
```

### 5. 临时解决方案

如果 Nacos 配置有问题，可以临时修改环境变量：

```bash
# 改为使用 public 命名空间
kubectl -n monitoring patch deployment domain-exporter -p '{"spec":{"template":{"spec":{"containers":[{"name":"domain-exporter","env":[{"name":"NACOS_NAMESPACE_ID","value":"public"}]}]}}}}'

# 或者完全禁用 Nacos
kubectl -n monitoring patch deployment domain-exporter -p '{"spec":{"template":{"spec":{"containers":[{"name":"domain-exporter","env":[{"name":"NACOS_URL","value":""}]}]}}}}'
```

## 🎯 推荐操作

1. **立即检查**：Pod 的环境变量配置
2. **验证网络**：Pod 到 Nacos 服务器的连通性
3. **确认配置**：Nacos 控制台中的命名空间和配置
4. **应用修复**：根据检查结果选择合适的修复方案

## 📝 调试命令

```bash
# 查看详细日志
kubectl -n monitoring logs -f domain-exporter-cbdf49596-7w854

# 查看配置端点
kubectl -n monitoring port-forward domain-exporter-cbdf49596-7w854 8080:8080
curl http://localhost:8080/config

# 查看指标
curl http://localhost:8080/metrics | grep domain_
```