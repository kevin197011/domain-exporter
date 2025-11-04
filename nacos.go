package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"gopkg.in/yaml.v2"
)

// NacosConfigManager Nacos配置管理器
type NacosConfigManager struct {
	client       config_client.IConfigClient
	config       *Config
	configMutex  sync.RWMutex
	updateChan   chan *Config
}

// NewNacosConfigManager 创建Nacos配置管理器
func NewNacosConfigManager(localConfig *Config) (*NacosConfigManager, error) {
	if !localConfig.IsNacosEnabled() {
		slog.Info("Nacos未启用，跳过Nacos配置管理器创建")
		return nil, nil
	}

	slog.Info("创建Nacos配置管理器", 
		"nacos_url", localConfig.NacosUrl,
		"namespace_id", localConfig.NamespaceId,
		"username", localConfig.Username,
		"data_id", localConfig.DataId,
		"group", localConfig.Group)

	// 解析 Nacos URL 获取主机和端口
	nacosURL := strings.TrimPrefix(localConfig.NacosUrl, "http://")
	nacosURL = strings.TrimPrefix(nacosURL, "https://")
	
	var host string
	var port uint64 = 8848 // 默认端口
	
	if strings.Contains(nacosURL, ":") {
		parts := strings.Split(nacosURL, ":")
		host = parts[0]
		if len(parts) > 1 {
			if p, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
				port = p
			}
		}
	} else {
		host = nacosURL
	}
	
	// 构建服务器配置
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: host,
			Port:   port,
		},
	}
	
	slog.Info("Nacos服务器配置", "host", host, "port", port)

	// 构建客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:         localConfig.NamespaceId,
		TimeoutMs:           20000, // 增加超时时间到20秒
		NotLoadCacheAtStart: true,  // 不从缓存启动，避免文件权限问题
		LogDir:              "/tmp/nacos/log",     // 使用临时目录
		CacheDir:            "/tmp/nacos/cache",   // 使用临时目录
		LogLevel:            "debug",  // 增加日志级别以便调试
		Username:            localConfig.Username,
		Password:            localConfig.Password,
		// Kubernetes环境优化配置
		UpdateThreadNum:      1,      // 减少线程数
		UpdateCacheWhenEmpty: false,  // 空配置时不更新缓存
		// 禁用一些可能导致问题的功能
		OpenKMS:             false,   // 禁用KMS
	}
	
	slog.Info("Nacos客户端配置", 
		"timeout_ms", clientConfig.TimeoutMs,
		"log_dir", clientConfig.LogDir,
		"cache_dir", clientConfig.CacheDir)

	// 为 HTTPS 连接配置 SSL 设置
	if strings.HasPrefix(localConfig.NacosUrl, "https://") {
		slog.Info("检测到HTTPS连接，配置SSL设置", "skip_ssl_verify", localConfig.SkipSSLVerify)
		
		if localConfig.SkipSSLVerify {
			// 全局设置跳过 SSL 证书验证（用于开发/测试环境）
			if transport, ok := http.DefaultTransport.(*http.Transport); ok {
				if transport.TLSClientConfig == nil {
					transport.TLSClientConfig = &tls.Config{}
				}
				transport.TLSClientConfig.InsecureSkipVerify = true
				slog.Warn("已禁用SSL证书验证，仅适用于开发/测试环境")
			}
		}
	}

	// 创建配置客户端
	slog.Info("正在创建Nacos配置客户端...", "host", host, "port", port)
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		slog.Error("创建Nacos配置客户端失败", "error", err)
		return nil, fmt.Errorf("创建Nacos配置客户端失败: %w", err)
	}
	
	slog.Info("Nacos配置客户端创建成功")

	manager := &NacosConfigManager{
		client:     client,
		config:     localConfig, // 先使用本地配置作为初始值
		updateChan: make(chan *Config, 1),
	}

	// 尝试从Nacos加载配置，增加重试机制
	maxRetries := 3
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		if err := manager.loadConfigFromNacos(); err != nil {
			lastErr = err
			slog.Warn("从Nacos加载配置失败", "attempt", i+1, "max_retries", maxRetries, "error", err)
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 2 * time.Second) // 递增延迟
			}
		} else {
			slog.Info("成功从Nacos加载配置", "attempt", i+1)
			lastErr = nil
			break
		}
	}
	
	if lastErr != nil {
		slog.Warn("多次尝试后仍无法从Nacos加载配置，将使用本地配置", "error", lastErr)
		// 不返回错误，继续使用本地配置
	}

	// 监听配置变化
	go manager.watchConfig()

	return manager, nil
}

// loadConfigFromNacos 从Nacos加载配置
func (m *NacosConfigManager) loadConfigFromNacos() error {
	slog.Info("尝试从Nacos加载配置", 
		"namespace", m.config.NamespaceId,
		"group", m.config.Group, 
		"data_id", m.config.DataId,
		"nacos_url", m.config.NacosUrl,
		"username", m.config.Username)
	
	// 添加详细的参数日志
	configParam := vo.ConfigParam{
		DataId: m.config.DataId,
		Group:  m.config.Group,
	}
	
	slog.Debug("Nacos请求参数", 
		"config_param", fmt.Sprintf("%+v", configParam))
		
	content, err := m.client.GetConfig(configParam)
	if err != nil {
		slog.Error("Nacos GetConfig 调用失败", 
			"error", err,
			"error_type", fmt.Sprintf("%T", err),
			"namespace", m.config.NamespaceId,
			"data_id", m.config.DataId,
			"group", m.config.Group)
		return fmt.Errorf("从Nacos获取配置失败: %w", err)
	}

	slog.Info("Nacos返回内容长度", "content_length", len(content))
	
	// 检查配置内容是否为空
	if content == "" {
		slog.Warn("Nacos配置内容为空", 
			"namespace", m.config.NamespaceId,
			"group", m.config.Group,
			"data_id", m.config.DataId,
			"nacos_console_url", fmt.Sprintf("%s/nacos", m.config.NacosUrl))
		return fmt.Errorf("Nacos配置内容为空，请在Nacos控制台创建配置: namespace=%s, group=%s, dataId=%s", 
			m.config.NamespaceId, m.config.Group, m.config.DataId)
	}
	
	slog.Debug("Nacos配置内容", "content", content)

	var nacosConfig Config
	if err := yaml.Unmarshal([]byte(content), &nacosConfig); err != nil {
		return err
	}

	// 保留原始的Nacos连接配置，不被覆盖
	nacosConfig.NacosUrl = m.config.NacosUrl
	nacosConfig.Username = m.config.Username
	nacosConfig.Password = m.config.Password
	nacosConfig.NamespaceId = m.config.NamespaceId
	nacosConfig.DataId = m.config.DataId
	nacosConfig.Group = m.config.Group

	// 应用默认值（只对未设置的值）
	applyDefaults(&nacosConfig)

	m.configMutex.Lock()
	m.config = &nacosConfig
	m.configMutex.Unlock()

	slog.Info("从Nacos成功加载配置", 
		"domain_count", len(nacosConfig.Domains),
		"check_interval", nacosConfig.CheckInterval,
		"timeout", nacosConfig.Timeout)

	// 通知配置更新
	select {
	case m.updateChan <- &nacosConfig:
		slog.Debug("已发送配置更新通知")
	default:
		slog.Warn("配置更新通道已满，跳过通知")
	}

	return nil
}

// watchConfig 监听配置变化
func (m *NacosConfigManager) watchConfig() {
	err := m.client.ListenConfig(vo.ConfigParam{
		DataId: m.config.DataId,
		Group:  m.config.Group,
		OnChange: func(namespace, group, dataId, data string) {
			slog.Info("检测到Nacos配置变化", "group", group, "data_id", dataId)
			
			if data == "" {
				slog.Warn("Nacos配置内容为空，忽略此次变更")
				return
			}
			
			var newConfig Config
			if err := yaml.Unmarshal([]byte(data), &newConfig); err != nil {
				slog.Error("解析Nacos配置失败", "error", err)
				return
			}

			// 保留原始的Nacos连接配置
			newConfig.NacosUrl = m.config.NacosUrl
			newConfig.Username = m.config.Username
			newConfig.Password = m.config.Password
			newConfig.NamespaceId = m.config.NamespaceId
			newConfig.DataId = m.config.DataId
			newConfig.Group = m.config.Group

			// 应用默认值（只对未设置的值）
			applyDefaults(&newConfig)

			m.configMutex.Lock()
			oldDomainCount := len(m.config.Domains)
			m.config = &newConfig
			m.configMutex.Unlock()

			slog.Info("Nacos配置已更新", 
				"old_domain_count", oldDomainCount, 
				"new_domain_count", len(newConfig.Domains),
				"check_interval", newConfig.CheckInterval,
				"timeout", newConfig.Timeout)

			// 通知配置更新
			select {
			case m.updateChan <- &newConfig:
				slog.Info("已发送Nacos配置变更通知")
			default:
				slog.Warn("配置更新通道已满，跳过通知")
			}
		},
	})

	if err != nil {
		slog.Error("监听Nacos配置变化失败", "error", err)
	}
}

// GetConfig 获取当前配置
func (m *NacosConfigManager) GetConfig() *Config {
	if m == nil {
		return nil
	}
	
	m.configMutex.RLock()
	defer m.configMutex.RUnlock()
	return m.config
}

// GetUpdateChannel 获取配置更新通道
func (m *NacosConfigManager) GetUpdateChannel() <-chan *Config {
	if m == nil {
		return nil
	}
	return m.updateChan
}

// Close 关闭Nacos客户端
func (m *NacosConfigManager) Close() {
	if m != nil && m.client != nil {
		// Nacos SDK没有显式的Close方法，这里只是占位
		close(m.updateChan)
	}
}

