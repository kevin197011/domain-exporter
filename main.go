package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configFile = flag.String("config", "config.yml", "配置文件路径")
	port       = flag.String("port", "8080", "HTTP服务端口")
)

func main() {
	flag.Parse()

	// 初始化结构化日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 加载配置
	config, err := LoadConfig(*configFile)
	if err != nil {
		slog.Error("加载配置文件失败", "error", err)
		os.Exit(1)
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
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		exporter.TriggerCheck()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "triggered", "message": "域名检查已触发"}`)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
		<html>
		<head><title>域名过期监控 Exporter</title></head>
		<body>
		<h1>域名过期监控 Exporter</h1>
		<p><a href="/metrics">Metrics</a></p>
		<p><a href="/trigger" onclick="triggerCheck(); return false;">手动触发检查</a></p>
		<script>
		function triggerCheck() {
			fetch('/trigger', {method: 'POST'})
				.then(response => response.json())
				.then(data => alert(data.message))
				.catch(error => alert('触发失败: ' + error));
		}
		</script>
		</body>
		</html>
		`)
	})

	// 启动HTTP服务
	serverPort := *port
	if config.Port != 0 {
		serverPort = fmt.Sprintf("%d", config.Port)
	}

	slog.Info("启动HTTP服务", "port", serverPort)
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: nil,
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		slog.Info("收到关闭信号，正在关闭服务...")
		exporter.Stop()
		server.Close()
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP服务启动失败", "error", err)
		os.Exit(1)
	}
}