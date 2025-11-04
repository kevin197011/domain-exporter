# 域名过期监控 Prometheus Exporter

这是一个用Go语言实现的Prometheus exporter，用于监控域名的过期时间。

## 功能特性

- 监控多个域名的过期时间
- 提供Prometheus格式的指标
- 支持配置文件
- 容器化部署
- 优雅关闭

## 指标说明

- `domain_expiry_days{domain="example.com"}` - 域名距离过期的天数 (-999表示检测失败)
- `domain_expiry_timestamp{domain="example.com"}` - 域名过期时间戳 (0表示检测失败)
- `domain_check_timestamp{domain="example.com"}` - 域名最后检查时间戳
- `domain_check_status{domain="example.com"}` - 域名检查状态 (1=成功, 0=失败)

## 安装和使用

### 本地运行

1. 安装依赖：
```bash
go mod tidy
```

2. 配置环境变量（创建 `.env` 文件或设置环境变量）：
```bash
# Nacos配置
NACOS_URL=http://127.0.0.1:8848
NACOS_USERNAME=nacos
NACOS_PASSWORD=nacos
NACOS_NAMESPACE_ID=public        # 可选，默认为 public
NACOS_DATA_ID=domain-exporter    # 可选，默认为 domain-exporter
NACOS_GROUP=DEFAULT_GROUP        # 可选，默认为 DEFAULT_GROUP
```

如果不使用Nacos，可以直接通过环境变量配置业务参数：
```bash
# 本地配置模式
DOMAINS=your-domain.com,another-domain.com
CHECK_INTERVAL=3600  # 检查间隔（秒）
PORT=8080           # HTTP服务端口
LOG_LEVEL=info      # 日志级别
TIMEOUT=30          # 请求超时时间（秒）
```

也可以继续使用配置文件（环境变量优先）：
```yaml
# config.yml（可选）
nacos_url: "http://127.0.0.1:8848"
username: "nacos"
password: "nacos"
```

3. 运行程序：
```bash
# 使用环境变量
export NACOS_URL=http://127.0.0.1:8848
export NACOS_USERNAME=nacos
export NACOS_PASSWORD=nacos
go run .

# 或者使用.env文件
go run .

# 或者使用配置文件
go run . -config=config.yml
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
# 修改.env文件中的配置
cp .env.example .env
# 编辑.env文件设置你的Nacos配置

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

#### 动态配置参数
所有以下参数都支持通过Nacos动态调整，无需重启服务：

- **domains**: 监控的域名列表，修改后立即触发检查
- **check_interval**: 检查间隔（秒），修改后在下次定时器触发时生效
- **port**: HTTP服务端口（注意：端口变更需要重启服务）
- **log_level**: 日志级别（debug/info/warn/error）

- **timeout**: WHOIS查询超时时间（秒），修改后在下次查询时生效

- **whois_servers**: 备用WHOIS服务器列表

#### 配置变更监控
- 访问 `http://localhost:8080/config` 查看当前配置
- 访问 `http://localhost:8080/metrics` 查看监控指标
- 修改Nacos配置后，系统会自动检测变化并记录日志

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
    expr: domain_expiry_days < 30 and domain_expiry_days > 0
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "域名即将过期"
      description: "域名 {{ $labels.domain }} 将在 {{ $value }} 天后过期"

  - alert: DomainCheckFailed
    expr: domain_check_status == 0 or domain_expiry_days == -999
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "域名检查失败"
      description: "无法获取域名 {{ $labels.domain }} 的过期信息，请检查域名状态"
```

### 错误码说明

- `domain_expiry_days = -999`: 表示域名检测失败，无法获取过期信息
- `domain_expiry_timestamp = 0`: 表示检测失败时的时间戳占位符
- `domain_check_status = 0`: 表示检查失败