# Prometheus 集成指南

## 🎯 ServiceMonitor 自动收集

### ✅ 自动收集条件

当 `serviceMonitor.enabled: true` 时，**Prometheus Operator** 会自动发现并收集指标，需要满足：

1. **Prometheus Operator 已安装**
2. **ServiceMonitor 标签匹配 Prometheus 选择器**
3. **网络连通性正常**

### 🔧 配置说明

```yaml
# helm/domain-exporter/values.yaml
serviceMonitor:
  enabled: true                    # 启用 ServiceMonitor
  namespace: ""                    # 留空使用当前命名空间
  labels: 
    release: prometheus            # 关键：匹配 Prometheus 选择器
  interval: 30s                    # 抓取间隔
  scrapeTimeout: 10s              # 抓取超时
  path: /metrics                   # 指标路径
```

## 🔍 验证 ServiceMonitor 是否生效

### 1. 检查 ServiceMonitor 资源

```bash
# 查看 ServiceMonitor 是否创建
kubectl -n monitoring get servicemonitor

# 查看详细配置
kubectl -n monitoring describe servicemonitor domain-exporter
```

### 2. 检查 Prometheus 是否发现

```bash
# 进入 Prometheus UI
kubectl -n monitoring port-forward svc/prometheus-server 9090:80

# 访问 http://localhost:9090
# 在 Status -> Targets 中查找 domain-exporter
```

### 3. 验证指标收集

```bash
# 在 Prometheus UI 中查询
domain_expiry_days
domain_check_status
```

## 🎛️ 不同 Prometheus 部署的配置

### kube-prometheus-stack

```yaml
serviceMonitor:
  enabled: true
  labels:
    release: prometheus  # 或者你的 Helm release 名称
```

### Prometheus Operator

```yaml
serviceMonitor:
  enabled: true
  labels:
    app: prometheus
    # 或者根据你的 Prometheus 配置
```

### 自定义标签

```bash
# 查看你的 Prometheus 配置
kubectl -n monitoring get prometheus -o yaml | grep -A 10 serviceMonitorSelector

# 根据输出配置对应的标签
```

## 📊 监控指标说明

### 核心指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `domain_expiry_days` | Gauge | 域名距离过期的天数 |
| `domain_expiry_timestamp` | Gauge | 域名过期时间戳 |
| `domain_check_timestamp` | Gauge | 最后检查时间戳 |
| `domain_check_status` | Gauge | 检查状态 (1=成功, 0=失败) |

### 标签维度

- `domain`: 域名名称
- `method`: 检查方法 (whois)

## 🚨 告警规则示例

```yaml
# prometheus-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: domain-expiry-rules
  labels:
    release: prometheus
spec:
  groups:
  - name: domain.rules
    rules:
    - alert: DomainExpiringSoon
      expr: domain_expiry_days < 30 and domain_expiry_days > 0
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "域名即将过期"
        description: "域名 {{ $labels.domain }} 将在 {{ $value }} 天后过期"
    
    - alert: DomainCheckFailed
      expr: domain_check_status == 0
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "域名检查失败"
        description: "无法获取域名 {{ $labels.domain }} 的过期信息"
```

## 📈 Grafana 仪表板

### 基础查询

```promql
# 域名过期天数
domain_expiry_days

# 即将过期的域名 (30天内)
domain_expiry_days < 30 and domain_expiry_days > 0

# 检查失败的域名
domain_check_status == 0

# 域名过期时间排序
sort(domain_expiry_days)
```

### 仪表板面板建议

1. **域名过期天数表格**
2. **即将过期域名列表**
3. **检查状态统计**
4. **过期时间趋势图**

## 🔧 故障排除

### ServiceMonitor 未被发现

1. **检查标签匹配**：
   ```bash
   kubectl -n monitoring get prometheus -o yaml | grep -A 5 serviceMonitorSelector
   ```

2. **检查命名空间**：
   ```bash
   # ServiceMonitor 和 Prometheus 是否在同一命名空间
   kubectl get servicemonitor -A
   ```

3. **检查 RBAC 权限**：
   ```bash
   kubectl -n monitoring get rolebinding,clusterrolebinding | grep prometheus
   ```

### 指标无法访问

1. **测试指标端点**：
   ```bash
   kubectl -n monitoring port-forward svc/domain-exporter 8080:8080
   curl http://localhost:8080/metrics
   ```

2. **检查网络策略**：
   ```bash
   kubectl -n monitoring get networkpolicy
   ```

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| Target 不出现 | 标签不匹配 | 检查 serviceMonitor.labels |
| 连接被拒绝 | 网络问题 | 检查 Service 和网络策略 |
| 指标为空 | 应用问题 | 检查应用日志和 /metrics 端点 |

## 🚀 部署命令

```bash
# 启用 ServiceMonitor 部署
helm upgrade domain-exporter ./helm/domain-exporter \
  -n monitoring \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.labels.release=prometheus

# 验证部署
kubectl -n monitoring get servicemonitor domain-exporter
```