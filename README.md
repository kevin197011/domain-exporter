# 域名过期监控 Prometheus Exporter

这是一个用Go语言实现的Prometheus exporter，用于监控域名的过期时间。

## 功能特性

- 监控多个域名的过期时间
- 提供Prometheus格式的指标
- 支持配置文件
- 容器化部署
- 优雅关闭

## 指标说明

- `domain_expiry_days{domain="example.com"}` - 域名距离过期的天数
- `domain_expiry_timestamp{domain="example.com"}` - 域名过期时间戳
- `domain_check_timestamp{domain="example.com"}` - 域名最后检查时间戳
- `domain_check_status{domain="example.com"}` - 域名检查状态 (1=成功, 0=失败)

## 安装和使用

### 本地运行

1. 安装依赖：
```bash
go mod tidy
```

2. 修改配置文件 `config.yml`：
```yaml
# Nacos配置 - 简化版本（如果不使用Nacos，可以留空）
nacos_url: "http://127.0.0.1:8848"
username: "nacos"
password: "nacos"
namespace_id: "public"        # 可选，默认为 public
data_id: "domain-exporter"    # 可选，默认为 domain-exporter
group: "DEFAULT_GROUP"        # 可选，默认为 DEFAULT_GROUP
```

如果不使用Nacos，可以直接在本地配置文件中添加业务配置：
```yaml
# 本地配置模式
domains:
  - your-domain.com
  - another-domain.com

check_interval: 3600  # 检查间隔（秒）
port: 8080           # HTTP服务端口
log_level: info      # 日志级别
max_concurrent: 5    # 最大并发检查数
timeout: 30          # 请求超时时间（秒）
```

3. 运行程序：
```bash
go run .
```

4. 访问指标：
```bash
curl http://localhost:8080/metrics
```

### 使用Nacos配置管理

1. 启动Nacos服务器
2. 在Nacos控制台创建配置：
   - Data ID: `domain-exporter`
   - Group: `DEFAULT_GROUP`
   - 配置内容参考 `nacos-config-example.yml`
3. 在本地配置文件中启用Nacos：`nacos.enabled: true`
4. 启动应用，配置将从Nacos动态加载
5. 在Nacos控制台修改配置，应用会自动重新加载

### Docker运行

#### 单独运行
1. 构建镜像：
```bash
docker build -t domain-exporter .
```

2. 运行容器：
```bash
docker run -d -p 8080:8080 -v $(pwd)/config.yml:/root/config.yml domain-exporter
```

#### 使用Docker Compose（包含Nacos）
```bash
# 启动Nacos + 域名监控
docker-compose up -d

# 启动完整监控栈（Nacos + 域名监控 + Prometheus + Grafana）
docker-compose -f docker-compose-full.yml up -d
```

#### Nacos配置步骤
1. 访问Nacos控制台：http://localhost:8848/nacos
2. 使用默认账户登录：用户名 `nacos`，密码 `nacos`
3. 创建配置：
   - Data ID: `domain-exporter`
   - Group: `DEFAULT_GROUP`
   - 配置格式: `YAML`
   - 配置内容参考 `nacos-config-example.yml`

## Prometheus配置

在Prometheus配置文件中添加：

```yaml
scrape_configs:
  - job_name: 'domain-exporter'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 60s
```

## Grafana仪表板

可以创建Grafana仪表板来可视化域名过期信息：

- 域名过期天数趋势图
- 即将过期的域名列表（如30天内）
- 域名检查状态

## 告警规则

可以设置Prometheus告警规则：

```yaml
groups:
- name: domain_expiry
  rules:
  - alert: DomainExpiringSoon
    expr: domain_expiry_days < 30
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "域名即将过期"
      description: "域名 {{ $labels.domain }} 将在 {{ $value }} 天后过期"

  - alert: DomainCheckFailed
    expr: domain_check_status == 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "域名检查失败"
      description: "无法获取域名 {{ $labels.domain }} 的过期信息"
```