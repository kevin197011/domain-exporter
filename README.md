# Domain Registration Expiry Checker Exporter

A domain registration expiry date checking tool implemented in Go, with Prometheus metrics export support.

## Features

- 🔍 **Domain Registration Expiry Check**: Automatically check domain registration expiry time via WHOIS queries
- 📊 **Prometheus Integration**: Export monitoring metrics in Prometheus format
- ⚙️ **YAML Configuration**: Configure domain lists and check parameters using YAML files
- 🚀 **Concurrency Control**: Support configurable concurrent check count
- ⏰ **Scheduled Checks**: Configurable check frequency
- 🌐 **Web Interface**: Provides simple web status page
- 💪 **Robustness**: Includes timeout control and error handling

## Quick Start

### 1. Install Dependencies

```bash
make deps
```

### 2. Configure Domains

Edit the `config.yaml` file and add domains to monitor:

```yaml
server:
  port: 8080
  metrics_path: "/metrics"

checker:
  check_interval: 3600  # Check interval (seconds)
  concurrency: 10       # Concurrency
  timeout: 30          # Connection timeout (seconds)

domains:
  - google.com
  - github.com
  - stackoverflow.com
  - example.com
```

### 3. Run Application

```bash
make run
```

Or run directly:

```bash
go run . -config=config.yaml
```

### 4. Access Services

- **Status Page**: http://localhost:8080
- **Prometheus Metrics**: http://localhost:8080/metrics
- **Health Check**: http://localhost:8080/health

## Configuration

### Server Configuration (server)

- `port`: HTTP service port, default 8080
- `metrics_path`: Prometheus metrics path, default "/metrics"

### Checker Configuration (checker)

- `check_interval`: Check interval time (seconds), default 3600 (1 hour)
- `concurrency`: Concurrent check count, default 10
- `timeout`: Connection timeout (seconds), default 30

### Domain Configuration (domains)

- Domain list: Use string array format directly, one domain per line

## Prometheus Metrics

This exporter exports the following metrics:

### domain_expiry_days
- **Type**: Gauge
- **Description**: Days remaining until domain registration expires
- **Labels**: 
  - `domain`: Domain name
  - `description`: Domain name (same as domain)

### domain_valid
- **Type**: Gauge  
- **Description**: Whether domain registration is valid (1=valid, 0=invalid)
- **Labels**:
  - `domain`: Domain name
  - `description`: Domain name (same as domain)
  - `error`: Error message (only when invalid)

### domain_last_check_timestamp
- **Type**: Gauge
- **Description**: Timestamp of last domain registration check
- **Labels**:
  - `domain`: Domain name
  - `description`: Domain name (same as domain)

## Build and Deploy

### Local Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Code Check

```bash
make check
```

## Usage Examples

### Prometheus Configuration

Add to Prometheus configuration file:

```yaml
scrape_configs:
  - job_name: 'domain-exporter'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 60s
```

### Grafana Alert Rules

```yaml
groups:
  - name: domain-expiry
    rules:
      - alert: DomainExpiringSoon
        expr: domain_expiry_days < 30
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Domain {{ $labels.domain }} registration expiring soon"
          description: "Domain {{ $labels.domain }} registration will expire in {{ $value }} days"
      
      - alert: DomainExpired
        expr: domain_expiry_days < 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Domain {{ $labels.domain }} registration has expired"
          description: "Domain {{ $labels.domain }} registration has expired"
```

## Project Structure

```
.
├── main.go              # Main program entry
├── config.yaml          # Configuration file
├── config/
│   └── config.go        # Configuration file parsing
├── checker/
│   └── domain_checker.go # Domain check logic
├── exporter/
│   └── metrics.go       # Prometheus metrics
├── Makefile            # Build script
└── README.md           # Project documentation
```

## License

MIT License