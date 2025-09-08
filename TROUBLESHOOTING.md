# Domain Exporter 故障排除指南

## 常见问题和解决方案

### 1. Kubernetes 部署问题

#### 问题：容器启动失败，提示找不到配置文件
```
Error: failed to create containerd task: failed to create shim task: OCI runtime create failed: runc create failed: unable to start container process: error during container init: exec: "-config=/etc/domain-exporter/config.yaml": stat -config=/etc/domain-exporter/config.yaml: no such file or directory
```

**解决方案**：
- 确保 Helm 模板中的配置路径正确
- 检查 ConfigMap 是否正确创建
- 验证 volumeMount 路径与应用程序参数一致

**验证步骤**：
```bash
# 1. 检查 Helm 模板渲染
helm template domain-exporter ./helm/domain-exporter | grep -A5 -B5 "args:\|mountPath:"

# 2. 检查 ConfigMap
kubectl get configmap domain-exporter-config -o yaml

# 3. 检查 Pod 状态
kubectl describe pod -l app.kubernetes.io/name=domain-exporter

# 4. 查看容器日志
kubectl logs -l app.kubernetes.io/name=domain-exporter
```

#### 问题：ConfigMap 未找到
```
Error: configmaps "domain-exporter-config" not found
```

**解决方案**：
```bash
# 重新安装 Helm chart
helm uninstall domain-exporter
helm install domain-exporter ./helm/domain-exporter

# 或者升级现有部署
helm upgrade domain-exporter ./helm/domain-exporter
```

### 2. 配置问题

#### 问题：域名检查失败
**症状**：所有域名显示为无效状态

**解决方案**：
1. 检查网络连接
2. 验证域名格式
3. 调整超时设置

```yaml
# config.yaml
checker:
  timeout: 60  # 增加超时时间
  concurrency: 5  # 减少并发数
```

#### 问题：检查频率过高
**症状**：频繁的 WHOIS 查询被限制

**解决方案**：
```yaml
# config.yaml
checker:
  check_interval: 7200  # 增加检查间隔到2小时
  concurrency: 3        # 减少并发数
```

### 3. Docker 部署问题

#### 问题：容器无法启动
```bash
# 检查容器日志
docker logs domain-exporter

# 检查配置文件挂载
docker exec domain-exporter ls -la /root/config.yaml

# 验证配置文件内容
docker exec domain-exporter cat /root/config.yaml
```

#### 问题：端口冲突
```bash
# 修改端口映射
docker-compose down
# 编辑 docker-compose.yml 修改端口
docker-compose up -d
```

### 4. 监控和指标问题

#### 问题：Prometheus 无法抓取指标
**检查步骤**：
```bash
# 1. 验证指标端点
curl http://localhost:8080/metrics

# 2. 检查 Prometheus 配置
# prometheus.yml
scrape_configs:
  - job_name: 'domain-exporter'
    static_configs:
      - targets: ['domain-exporter:8080']  # 在 Docker 网络中使用服务名
```

#### 问题：指标数据不更新
**可能原因**：
- 域名检查失败
- 网络连接问题
- 配置错误

**解决方案**：
```bash
# 查看应用日志
kubectl logs -l app.kubernetes.io/name=domain-exporter -f

# 检查域名状态页面
curl http://localhost:8080/
```

### 5. 性能优化

#### 大量域名监控优化
```yaml
# config.yaml
checker:
  check_interval: 3600    # 1小时检查一次
  concurrency: 20         # 增加并发数
  timeout: 45            # 适中的超时时间

# 分批监控不同类型的域名
domains:
  - critical-domain1.com
  - critical-domain2.com
  # ... 重要域名放在前面
```

### 6. 调试命令

#### 本地调试
```bash
# 构建和测试
make build
make test-app

# 查看详细日志
go run . -config=config.yaml
```

#### Kubernetes 调试
```bash
# 查看 Pod 详情
kubectl describe pod -l app.kubernetes.io/name=domain-exporter

# 进入容器调试
kubectl exec -it deployment/domain-exporter -- sh

# 查看配置文件
kubectl exec deployment/domain-exporter -- cat /root/config.yaml

# 实时查看日志
kubectl logs -l app.kubernetes.io/name=domain-exporter -f
```

#### Helm 调试
```bash
# 验证 Chart
make test-helm

# 渲染模板查看生成的 YAML
helm template domain-exporter ./helm/domain-exporter > debug.yaml

# 检查 Chart 语法
helm lint ./helm/domain-exporter
```

### 7. 常用检查清单

#### 部署前检查
- [ ] 配置文件格式正确
- [ ] 域名列表有效
- [ ] 网络连接正常
- [ ] 资源限制合理

#### 部署后检查
- [ ] Pod 状态为 Running
- [ ] 健康检查通过
- [ ] 指标端点可访问
- [ ] 域名检查正常工作

#### 监控检查
- [ ] Prometheus 能抓取指标
- [ ] Grafana 仪表板显示数据
- [ ] 告警规则配置正确
- [ ] 通知渠道工作正常

### 8. 联系支持

如果问题仍然存在，请提供以下信息：
- 错误日志
- 配置文件内容
- 部署环境信息
- 复现步骤

```bash
# 收集调试信息
kubectl get all -l app.kubernetes.io/name=domain-exporter
kubectl describe pod -l app.kubernetes.io/name=domain-exporter
kubectl logs -l app.kubernetes.io/name=domain-exporter --tail=100
```