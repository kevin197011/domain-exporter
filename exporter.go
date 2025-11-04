package main

import (
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// DomainExporter Prometheus exporter结构
type DomainExporter struct {
	config      *Config
	mutex       sync.RWMutex
	nacosManager *NacosConfigManager
	stopChan    chan struct{}
	triggerChan chan struct{} // 用于触发立即检查

	// Prometheus指标
	domainExpiryDays *prometheus.GaugeVec
	domainExpiryTime *prometheus.GaugeVec
	domainCheckTime  *prometheus.GaugeVec
	domainStatus     *prometheus.GaugeVec
}

// NewDomainExporter 创建新的exporter
func NewDomainExporter(localConfig *Config) (*DomainExporter, error) {
	var finalConfig *Config
	var nacosManager *NacosConfigManager
	
	// 如果启用了Nacos，优先尝试从Nacos获取配置
	if localConfig.IsNacosEnabled() {
		var err error
		nacosManager, err = NewNacosConfigManager(localConfig)
		if err != nil {
			slog.Warn("创建Nacos配置管理器失败，使用本地配置", "error", err)
			finalConfig = localConfig
		} else {
			// 尝试从Nacos获取配置
			if nacosConfig := nacosManager.GetConfig(); nacosConfig != nil {
				finalConfig = nacosConfig
				slog.Info("使用Nacos配置", "domain_count", len(nacosConfig.Domains))
			} else {
				slog.Info("Nacos配置为空，使用本地配置")
				finalConfig = localConfig
			}
		}
	} else {
		slog.Info("Nacos未启用，使用本地配置")
		finalConfig = localConfig
	}

	exporter := &DomainExporter{
		config:       finalConfig,
		nacosManager: nacosManager,
		stopChan:     make(chan struct{}),
		triggerChan:  make(chan struct{}, 1), // 缓冲通道，避免阻塞
		domainExpiryDays: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "domain_expiry_days",
				Help: "域名距离过期的天数",
			},
			[]string{"domain"},
		),
		domainExpiryTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "domain_expiry_timestamp",
				Help: "域名过期时间戳",
			},
			[]string{"domain"},
		),
		domainCheckTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "domain_check_timestamp",
				Help: "域名最后检查时间戳",
			},
			[]string{"domain"},
		),
		domainStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "domain_check_status",
				Help: "域名检查状态 (1=成功, 0=失败)",
			},
			[]string{"domain"},
		),
	}

	// 启动配置监听
	if nacosManager != nil {
		go exporter.watchConfigUpdates()
	}

	return exporter, nil
}

// Describe 实现Prometheus Collector接口
func (e *DomainExporter) Describe(ch chan<- *prometheus.Desc) {
	e.domainExpiryDays.Describe(ch)
	e.domainExpiryTime.Describe(ch)
	e.domainCheckTime.Describe(ch)
	e.domainStatus.Describe(ch)
}

// Collect 实现Prometheus Collector接口
func (e *DomainExporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	e.domainExpiryDays.Collect(ch)
	e.domainExpiryTime.Collect(ch)
	e.domainCheckTime.Collect(ch)
	e.domainStatus.Collect(ch)
}

// StartMonitoring 启动后台监控
func (e *DomainExporter) StartMonitoring() {
	// 立即执行一次检查
	e.checkAllDomains()

	// 定时检查
	ticker := time.NewTicker(time.Duration(e.getCurrentConfig().CheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.checkAllDomains()
		case <-e.triggerChan:
			slog.Info("收到配置变更触发信号，立即执行域名检查")
			e.checkAllDomains()
			// 重置定时器，避免频繁检查
			ticker.Reset(time.Duration(e.getCurrentConfig().CheckInterval) * time.Second)
		case <-e.stopChan:
			return
		}
	}
}

// watchConfigUpdates 监听配置更新
func (e *DomainExporter) watchConfigUpdates() {
	if e.nacosManager == nil {
		return
	}

	updateChan := e.nacosManager.GetUpdateChannel()
	for {
		select {
		case newConfig := <-updateChan:
			if newConfig != nil {
				e.mutex.Lock()
				oldDomainCount := len(e.config.Domains)
				oldDomains := make([]string, len(e.config.Domains))
				copy(oldDomains, e.config.Domains)
				
				e.config = newConfig
				newDomainCount := len(newConfig.Domains)
				e.mutex.Unlock()
				
				slog.Info("Nacos配置已更新", 
					"old_domain_count", oldDomainCount, 
					"new_domain_count", newDomainCount)
				
				// 检查域名列表是否有变化
				domainsChanged := e.checkDomainsChanged(oldDomains, newConfig.Domains)
				if domainsChanged {
					slog.Info("检测到域名列表变化，立即触发检查")
				} else {
					slog.Info("域名列表未变化，但其他配置可能已更新，立即触发检查")
				}
				
				// 触发立即检查
				select {
				case e.triggerChan <- struct{}{}:
				default:
					// 如果通道已满，说明已经有待处理的触发信号，跳过
				}
			}
		case <-e.stopChan:
			return
		}
	}
}

// checkDomainsChanged 检查域名列表是否有变化
func (e *DomainExporter) checkDomainsChanged(oldDomains, newDomains []string) bool {
	if len(oldDomains) != len(newDomains) {
		return true
	}
	
	// 创建map用于快速查找
	oldMap := make(map[string]bool)
	for _, domain := range oldDomains {
		oldMap[domain] = true
	}
	
	// 检查新域名列表中是否有不在旧列表中的域名
	for _, domain := range newDomains {
		if !oldMap[domain] {
			return true
		}
	}
	
	return false
}

// getCurrentConfig 获取当前配置
func (e *DomainExporter) getCurrentConfig() *Config {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.config
}

// Stop 停止监控
func (e *DomainExporter) Stop() {
	close(e.stopChan)
	if e.nacosManager != nil {
		e.nacosManager.Close()
	}
}

// TriggerCheck 手动触发检查（用于外部调用）
func (e *DomainExporter) TriggerCheck() {
	select {
	case e.triggerChan <- struct{}{}:
		slog.Info("手动触发域名检查")
	default:
		slog.Info("检查已在进行中，跳过手动触发")
	}
}

// checkAllDomains 检查所有域名
func (e *DomainExporter) checkAllDomains() {
	currentConfig := e.getCurrentConfig()
	slog.Info("开始检查域名", 
		"domain_count", len(currentConfig.Domains), 
		"max_concurrent", currentConfig.MaxConcurrent)

	// 创建信号量控制并发数
	semaphore := make(chan struct{}, currentConfig.MaxConcurrent)
	var wg sync.WaitGroup

	for _, domain := range currentConfig.Domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			e.checkDomain(d)
		}(domain)
	}

	// 等待所有检查完成
	wg.Wait()
	slog.Info("所有域名检查完成")
}

// checkDomain 检查单个域名
func (e *DomainExporter) checkDomain(domain string) {
	slog.Debug("检查域名", "domain", domain)

	// 记录检查时间
	now := time.Now()
	e.domainCheckTime.WithLabelValues(domain).Set(float64(now.Unix()))

	// 获取当前配置
	currentConfig := e.getCurrentConfig()

	// 获取域名信息（带超时和多种检测方法）
	timeout := time.Duration(currentConfig.Timeout) * time.Second
	domainInfo, err := GetDomainInfoWithFallback(domain, timeout, currentConfig)
	if err != nil {
		slog.Error("获取域名信息失败", "domain", domain, "error", err)
		e.domainStatus.WithLabelValues(domain).Set(0)
		return
	}

	// 设置成功状态
	e.domainStatus.WithLabelValues(domain).Set(1)

	// 计算剩余天数（取整数）
	daysUntilExpiry := time.Until(domainInfo.ExpiryDate).Hours() / 24
	daysUntilExpiryInt := float64(int(daysUntilExpiry))
	e.domainExpiryDays.WithLabelValues(domain).Set(daysUntilExpiryInt)

	// 设置过期时间戳
	e.domainExpiryTime.WithLabelValues(domain).Set(float64(domainInfo.ExpiryDate.Unix()))

	slog.Info("域名检查完成", 
		"domain", domain,
		"days_until_expiry", int(daysUntilExpiryInt),
		"expiry_date", domainInfo.ExpiryDate.Format("2006-01-02"),
		"method", domainInfo.Method)
}