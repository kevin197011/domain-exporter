package main

import (
	"io/ioutil"
	"strconv"
	"strings"
	"gopkg.in/yaml.v2"
)

// Config 配置结构
type Config struct {
	// 业务配置（从Nacos获取）
	Domains           []string `yaml:"domains"`
	CheckInterval     int      `yaml:"check_interval"`
	Port              int      `yaml:"port"`
	LogLevel          string   `yaml:"log_level"`
	MaxConcurrent     int      `yaml:"max_concurrent"`
	Timeout           int      `yaml:"timeout"`
	DetectionMethods  []string `yaml:"detection_methods"`
	WhoisServers      []string `yaml:"whois_servers"`
	
	// Nacos连接配置（从本地配置文件获取）
	NacosUrl      string `yaml:"nacos_url"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	NamespaceId   string `yaml:"namespace_id"`
	DataId        string `yaml:"data_id"`
	Group         string `yaml:"group"`
}

// LoadConfig 加载配置文件
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	if config.CheckInterval == 0 {
		config.CheckInterval = 3600 // 默认1小时
	}
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 5 // 默认并发数5
	}
	if config.Timeout == 0 {
		config.Timeout = 30 // 默认超时30秒
	}
	if len(config.DetectionMethods) == 0 {
		config.DetectionMethods = []string{"whois"} // 默认检测方法
	}
	if len(config.WhoisServers) == 0 {
		config.WhoisServers = []string{
			"whois.verisign-grs.com",
			"whois.internic.net", 
			"whois.iana.org",
			"whois.publicinterestregistry.net",
			"whois.afilias.net",
		}
	}
	
	// 设置Nacos默认值
	if config.DataId == "" {
		config.DataId = "domain-exporter"
	}
	if config.Group == "" {
		config.Group = "DEFAULT_GROUP"
	}
	if config.NamespaceId == "" {
		config.NamespaceId = "public"
	}

	return &config, nil
}

// IsNacosEnabled 检查是否启用了Nacos
func (c *Config) IsNacosEnabled() bool {
	return c.NacosUrl != ""
}

// GetNacosServerHost 从URL中提取服务器地址
func (c *Config) GetNacosServerHost() string {
	if c.NacosUrl == "" {
		return ""
	}
	
	// 简单解析URL，提取主机和端口
	url := c.NacosUrl
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}
	
	// 移除路径部分
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	
	return url
}

// GetNacosServerIP 获取Nacos服务器IP
func (c *Config) GetNacosServerIP() string {
	host := c.GetNacosServerHost()
	if host == "" {
		return "127.0.0.1"
	}
	
	// 分离IP和端口
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	
	return host
}

// GetNacosServerPort 获取Nacos服务器端口
func (c *Config) GetNacosServerPort() uint64 {
	host := c.GetNacosServerHost()
	if host == "" {
		return 8848
	}
	
	// 分离IP和端口
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		portStr := host[idx+1:]
		if port, err := strconv.ParseUint(portStr, 10, 64); err == nil {
			return port
		}
	}
	
	return 8848
}