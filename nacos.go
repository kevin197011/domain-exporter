package main

import (
	"fmt"
	"log/slog"
	"sync"

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
		return nil, nil
	}

	// 构建服务器配置
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: localConfig.GetNacosServerIP(),
			Port:   localConfig.GetNacosServerPort(),
		},
	}

	// 构建客户端配置，使用默认值
	clientConfig := constant.ClientConfig{
		NamespaceId:         localConfig.NamespaceId,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "info",
		Username:            localConfig.Username,
		Password:            localConfig.Password,
	}

	// 创建配置客户端
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}

	manager := &NacosConfigManager{
		client:     client,
		config:     localConfig, // 先使用本地配置作为初始值
		updateChan: make(chan *Config, 1),
	}

	// 尝试从Nacos加载配置，如果失败则保持使用本地配置
	if err := manager.loadConfigFromNacos(); err != nil {
		slog.Warn("从Nacos加载配置失败，将使用本地配置", "error", err)
		// 不返回错误，继续使用本地配置
	} else {
		slog.Info("成功从Nacos加载配置")
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
		"data_id", m.config.DataId)
		
	content, err := m.client.GetConfig(vo.ConfigParam{
		DataId: m.config.DataId,
		Group:  m.config.Group,
	})
	if err != nil {
		return fmt.Errorf("从Nacos获取配置失败: %w", err)
	}

	// 检查配置内容是否为空
	if content == "" {
		return fmt.Errorf("Nacos配置内容为空，请在Nacos控制台创建配置: namespace=%s, group=%s, dataId=%s", 
			m.config.NamespaceId, m.config.Group, m.config.DataId)
	}

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