package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

// DomainInfo 域名信息结构
type DomainInfo struct {
	Domain     string
	ExpiryDate time.Time
	Registrar  string
	Status     string
	Method     string // 检测方法: whois
}

// GetDomainInfo 获取域名信息
func GetDomainInfo(domain string, timeout time.Duration) (*DomainInfo, error) {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 使用channel来处理超时
	type result struct {
		data string
		err  error
	}

	resultChan := make(chan result, 1)
	
	// 在goroutine中执行whois查询
	go func() {
		data, err := whois.Whois(domain)
		resultChan <- result{data: data, err: err}
	}()

	// 等待结果或超时
	select {
	case res := <-resultChan:
		if res.err != nil {
			return nil, fmt.Errorf("whois查询失败: %v", res.err)
		}
		return parseDomainInfo(domain, res.data)
	case <-ctx.Done():
		return nil, fmt.Errorf("whois查询超时: %v", ctx.Err())
	}
}

// parseDomainInfo 解析域名信息
func parseDomainInfo(domain, whoisData string) (*DomainInfo, error) {

	// 解析whois信息
	parsed, err := whoisparser.Parse(whoisData)
	if err != nil {
		return nil, fmt.Errorf("whois解析失败: %v", err)
	}

	// 解析过期时间
	expiryDate, err := time.Parse("2006-01-02T15:04:05Z", parsed.Domain.ExpirationDate)
	if err != nil {
		// 尝试其他时间格式
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			"02-Jan-2006",
			"2006/01/02",
		}
		
		for _, format := range formats {
			if expiryDate, err = time.Parse(format, parsed.Domain.ExpirationDate); err == nil {
				break
			}
		}
		
		if err != nil {
			return nil, fmt.Errorf("无法解析过期时间: %s", parsed.Domain.ExpirationDate)
		}
	}

	// 安全获取域名状态
	var status string
	if len(parsed.Domain.Status) > 0 {
		status = parsed.Domain.Status[0]
	} else {
		status = "unknown"
	}

	return &DomainInfo{
		Domain:     domain,
		ExpiryDate: expiryDate,
		Registrar:  parsed.Registrar.Name,
		Status:     status,
		Method:     "whois",
	}, nil
}

// GetDomainInfoWithFallback 使用WHOIS和备用服务器获取域名信息
func GetDomainInfoWithFallback(domain string, timeout time.Duration, config *Config) (*DomainInfo, error) {
	return GetDomainInfoWithSmartFallback(domain, timeout, config)
}

// GetDomainInfoWithBackupServers 使用备用WHOIS服务器
func GetDomainInfoWithBackupServers(domain string, timeout time.Duration, servers []string) (*DomainInfo, error) {
	var lastErr error
	
	for i, server := range servers {
		log.Printf("尝试备用WHOIS服务器 %d/%d: %s", i+1, len(servers), server)
		info, err := queryWhoisServer(domain, server, timeout)
		if err == nil {
			log.Printf("备用服务器 %s 查询成功", server)
			return info, nil
		}
		log.Printf("备用服务器 %s 查询失败: %v", server, err)
		lastErr = err
	}
	return nil, fmt.Errorf("所有备用WHOIS服务器都无法访问，最后错误: %v", lastErr)
}

// queryWhoisServer 查询指定的WHOIS服务器
func queryWhoisServer(domain, server string, timeout time.Duration) (*DomainInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// 连接到WHOIS服务器
	conn, err := net.DialTimeout("tcp", server+":43", timeout/2) // 连接超时设为总超时的一半
	if err != nil {
		return nil, fmt.Errorf("连接WHOIS服务器 %s 失败: %v", server, err)
	}
	defer conn.Close()
	
	// 发送查询
	_, err = conn.Write([]byte(domain + "\r\n"))
	if err != nil {
		return nil, fmt.Errorf("向服务器 %s 发送查询失败: %v", server, err)
	}
	
	// 读取响应
	buffer := make([]byte, 8192) // 增加缓冲区大小
	var response strings.Builder
	readTimeout := time.Second * 3 // 读取超时
	
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("从服务器 %s 查询超时", server)
		default:
			conn.SetReadDeadline(time.Now().Add(readTimeout))
			n, err := conn.Read(buffer)
			if err != nil {
				if n == 0 {
					// 读取完成
					break
				}
				// 检查是否是超时错误
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// 如果已经读取到一些数据，继续处理
					if response.Len() > 0 {
						break
					}
				}
				return nil, fmt.Errorf("从服务器 %s 读取响应失败: %v", server, err)
			}
			response.Write(buffer[:n])
			if n < len(buffer) {
				break
			}
		}
	}
	
	responseData := response.String()
	if len(responseData) == 0 {
		return nil, fmt.Errorf("从服务器 %s 获得空响应", server)
	}
	
	// 解析响应
	info, err := parseDomainInfo(domain, responseData)
	if err != nil {
		return nil, fmt.Errorf("解析服务器 %s 响应失败: %v", server, err)
	}
	
	// 标记使用的服务器
	info.Method = fmt.Sprintf("whois(%s)", server)
	
	return info, nil
}

// getPreferredWhoisServers 根据域名后缀获取首选的WHOIS服务器
func getPreferredWhoisServers(domain string) []string {
	// 提取域名后缀
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return []string{}
	}
	
	tld := strings.ToLower(parts[len(parts)-1])
	
	// 根据TLD返回首选服务器
	switch tld {
	case "com", "net":
		return []string{"whois.verisign-grs.com"}
	case "org":
		return []string{"whois.publicinterestregistry.net"}
	case "info":
		return []string{"whois.afilias.net"}
	case "biz":
		return []string{"whois.neulevel.biz"}
	case "us":
		return []string{"whois.nic.us"}
	case "uk":
		return []string{"whois.nominet.uk"}
	case "de":
		return []string{"whois.denic.de"}
	case "fr":
		return []string{"whois.afnic.fr"}
	case "jp":
		return []string{"whois.jprs.jp"}
	case "cn":
		return []string{"whois.cnnic.cn"}
	default:
		return []string{}
	}
}

// GetDomainInfoWithSmartFallback 使用智能备用服务器获取域名信息
func GetDomainInfoWithSmartFallback(domain string, timeout time.Duration, config *Config) (*DomainInfo, error) {
	// 首先尝试标准WHOIS查询
	info, err := GetDomainInfo(domain, timeout)
	if err == nil {
		return info, nil
	}
	
	log.Printf("标准WHOIS查询失败: %v，尝试智能备用服务器", err)
	
	// 获取针对该域名的首选服务器
	preferredServers := getPreferredWhoisServers(domain)
	if len(preferredServers) > 0 {
		log.Printf("尝试针对域名 %s 的首选服务器", domain)
		info, err = GetDomainInfoWithBackupServers(domain, timeout, preferredServers)
		if err == nil {
			return info, nil
		}
		log.Printf("首选服务器查询失败: %v", err)
	}
	
	// 如果首选服务器失败，尝试通用备用服务器
	if len(config.WhoisServers) > 0 {
		log.Printf("尝试通用备用服务器")
		info, err = GetDomainInfoWithBackupServers(domain, timeout, config.WhoisServers)
		if err == nil {
			return info, nil
		}
		log.Printf("通用备用服务器查询失败: %v", err)
	}
	
	return nil, fmt.Errorf("所有WHOIS查询方法都失败了")
}