package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"kevin197011.github.io/domain-exporter/checker"
	"kevin197011.github.io/domain-exporter/config"
	"kevin197011.github.io/domain-exporter/exporter"
)

func main() {
	var configPath = flag.String("config", "config.yaml", "Configuration file path")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully, will monitor %d domain registration expiry times", len(cfg.Domains))
	log.Printf("Check interval: %d seconds", cfg.Checker.CheckInterval)
	log.Printf("Concurrency: %d", cfg.Checker.Concurrency)
	log.Printf("Timeout: %d seconds", cfg.Checker.Timeout)

	// Create domain checker
	domainChecker := checker.NewDomainChecker(cfg)
	
	// Create metrics collector
	metrics := exporter.NewMetrics()
	metrics.Register()

	// Start domain registration checker
	domainChecker.Start()

	// Periodically update metrics
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Update metrics every 30 seconds
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				domainInfos := domainChecker.GetDomainInfos()
				metrics.UpdateMetrics(domainInfos)
			}
		}
	}()

	// Set up HTTP routes
	http.Handle(cfg.Server.MetricsPath, promhttp.Handler())
	
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Status page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		domainInfos := domainChecker.GetDomainInfos()
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<html><head><title>Domain Expiry Checker</title></head><body>")
		fmt.Fprintf(w, "<h1>Domain Registration Expiry Checker</h1>")
		fmt.Fprintf(w, "<p>Monitoring %d domain registration expiry times</p>", len(domainInfos))
		fmt.Fprintf(w, "<table border='1' style='border-collapse: collapse;'>")
		fmt.Fprintf(w, "<tr><th>Domain</th><th>Description</th><th>Status</th><th>Days Left</th><th>Registration Expiry</th><th>Last Check</th></tr>")
		
		for _, info := range domainInfos {
			status := "❌ Invalid"
			if info.IsValid {
				status = "✅ Valid"
			}
			
			expiryStr := "N/A"
			if info.IsValid {
				expiryStr = info.ExpiryDate.Format("2006-01-02 15:04:05")
			}
			
			fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td></tr>",
				info.Name, info.Description, status, info.DaysLeft, expiryStr, info.LastCheck.Format("2006-01-02 15:04:05"))
		}
		
		fmt.Fprintf(w, "</table>")
		fmt.Fprintf(w, "<p><a href='%s'>Prometheus Metrics</a></p>", cfg.Server.MetricsPath)
		fmt.Fprintf(w, "</body></html>")
	})

	// Start HTTP server
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting HTTP server, listening on port %d", cfg.Server.Port)
	log.Printf("Visit http://localhost:%d to view status", cfg.Server.Port)
	log.Printf("Visit http://localhost:%d%s to view Prometheus metrics", cfg.Server.Port, cfg.Server.MetricsPath)

	server := &http.Server{
		Addr:         serverAddr,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		log.Println("Received shutdown signal, gracefully shutting down...")
		server.Close()
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server failed to start: %v", err)
	}
	
	log.Println("Server has been shut down")
}