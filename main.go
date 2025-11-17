package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configFile = flag.String("config", "", "配置文件路径（可选，优先使用环境变量）")
	port       = flag.String("port", "", "HTTP服务端口（可选，优先使用环境变量）")
)

func main() {
	flag.Parse()

	// 加载配置
	config, err := LoadConfig(*configFile)
	if err != nil {
		slog.Error("加载配置文件失败", "error", err)
		os.Exit(1)
	}

	// 设置日志级别
	logLevel := parseLogLevel(config.LogLevel)
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		logLevel = parseLogLevel(envLogLevel)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// 打印详细的配置信息用于调试
	slog.Info("配置加载完成",
		"domains", len(config.Domains),
		"check_interval", config.CheckInterval,
		"port", config.Port,
		"timeout", config.Timeout,
		"nacos_enabled", config.IsNacosEnabled())

	// 如果启用了Nacos，打印详细的Nacos配置
	if config.IsNacosEnabled() {
		slog.Info("Nacos配置详情",
			"nacos_url", config.NacosUrl,
			"username", config.Username,
			"namespace_id", config.NamespaceId,
			"data_id", config.DataId,
			"group", config.Group)

		// 打印环境变量以便调试
		slog.Debug("环境变量调试信息",
			"NACOS_URL", os.Getenv("NACOS_URL"),
			"NACOS_USERNAME", os.Getenv("NACOS_USERNAME"),
			"NACOS_NAMESPACE_ID", os.Getenv("NACOS_NAMESPACE_ID"),
			"NACOS_DATA_ID", os.Getenv("NACOS_DATA_ID"),
			"NACOS_GROUP", os.Getenv("NACOS_GROUP"))
	}

	// 创建exporter
	exporter, err := NewDomainExporter(config)
	if err != nil {
		slog.Error("创建exporter失败", "error", err)
		os.Exit(1)
	}

	// 注册Prometheus指标
	prometheus.MustRegister(exporter)

	// 启动后台监控
	go exporter.StartMonitoring()

	// 设置HTTP路由
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		exporter.TriggerCheck()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "triggered",
			"message": "域名检查已触发",
		})
	})
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		currentConfig := exporter.getCurrentConfig()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		configJSON := map[string]interface{}{
			"domains":          currentConfig.Domains,
			"domain_count":     len(currentConfig.Domains),
			"check_interval":   currentConfig.CheckInterval,
			"port":             currentConfig.Port,
			"log_level":        currentConfig.LogLevel,
			"timeout":          currentConfig.Timeout,
			"detection_method": "whois",
			"execution_mode":   "serial",
			"nacos_enabled":    currentConfig.IsNacosEnabled(),
			"nacos_url":        currentConfig.NacosUrl,
			"nacos_namespace":  currentConfig.NamespaceId,
			"nacos_data_id":    currentConfig.DataId,
			"nacos_group":      currentConfig.Group,
		}
		json.NewEncoder(w).Encode(configJSON)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="zh-CN">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>域名过期监控 Exporter</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 40px; }
		h1 { color: #333; }
		.button {
			display: inline-block;
			padding: 10px 20px;
			margin: 10px 5px;
			background-color: #007cba;
			color: white;
			text-decoration: none;
			border-radius: 5px;
			border: none;
			cursor: pointer;
		}
		.button:hover { background-color: #005a87; }
		.info { margin-top: 20px; padding: 15px; background-color: #f5f5f5; border-radius: 5px; }
	</style>
</head>
<body>
	<h1>域名过期监控 Exporter</h1>
	<div>
		<a href="/metrics" class="button">查看 Metrics</a>
		<button onclick="triggerCheck()" class="button">手动触发检查</button>
		<a href="/config" class="button">查看配置</a>
	</div>
	<div class="info">
		<h3>功能说明</h3>
		<ul>
			<li><strong>Metrics</strong>: Prometheus 格式的监控指标</li>
			<li><strong>手动触发检查</strong>: 立即执行一次域名过期检查</li>
			<li><strong>查看配置</strong>: 显示当前的配置信息</li>
		</ul>
	</div>
	<script>
	function triggerCheck() {
		fetch('/trigger', {method: 'POST'})
			.then(response => response.json())
			.then(data => {
				alert('✅ ' + data.message);
			})
			.catch(error => {
				alert('❌ 触发失败: ' + error);
			});
	}
	</script>
</body>
</html>`)
	})

	// 启动HTTP服务
	serverPort := *port
	if serverPort == "" {
		if config.Port != 0 {
			serverPort = fmt.Sprintf("%d", config.Port)
		} else {
			serverPort = "8080" // 默认端口
		}
	}

	slog.Info("启动HTTP服务", "port", serverPort)
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: mux,
	}

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("收到关闭信号，正在关闭服务...")

		// 停止 exporter
		exporter.Stop()

		// 优雅关闭 HTTP 服务
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("HTTP服务关闭失败", "error", err)
		} else {
			slog.Info("HTTP服务已优雅关闭")
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP服务启动失败", "error", err)
		os.Exit(1)
	}
}

// parseLogLevel 解析日志级别字符串
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
