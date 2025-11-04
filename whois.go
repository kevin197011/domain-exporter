package main

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
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
	slog.Debug("开始标准WHOIS查询", "domain", domain, "timeout", timeout)
	
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
		slog.Debug("执行WHOIS查询", "domain", domain)
		data, err := whois.Whois(domain)
		if err != nil {
			slog.Debug("WHOIS查询失败", "domain", domain, "error", err)
		} else {
			slog.Debug("WHOIS查询成功", "domain", domain, "data_length", len(data))
		}
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
		slog.Debug("WHOIS查询超时", "domain", domain, "timeout", timeout)
		return nil, fmt.Errorf("whois查询超时: %v", ctx.Err())
	}
}

// parseDomainInfo 解析域名信息
func parseDomainInfo(domain, whoisData string) (*DomainInfo, error) {
	slog.Debug("开始解析WHOIS数据", "domain", domain, "data_length", len(whoisData))
	
	// 打印WHOIS原始数据的前500字符用于调试
	if len(whoisData) > 0 {
		preview := whoisData
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		slog.Debug("WHOIS原始数据预览", "domain", domain, "data", preview)
	}

	// 解析whois信息
	parsed, err := whoisparser.Parse(whoisData)
	if err != nil {
		slog.Error("WHOIS解析失败", "domain", domain, "error", err, "raw_data_length", len(whoisData))
		return nil, fmt.Errorf("whois解析失败: %v", err)
	}
	
	slog.Debug("WHOIS解析成功", "domain", domain, 
		"registrar", parsed.Registrar.Name,
		"expiration_date", parsed.Domain.ExpirationDate,
		"status_count", len(parsed.Domain.Status))

	// 检查解析结果
	if parsed.Domain.ExpirationDate == "" {
		slog.Error("WHOIS解析结果中没有过期时间", "domain", domain, 
			"registrar", parsed.Registrar.Name,
			"domain_name", parsed.Domain.Name)
		
		// 尝试从原始数据中手动提取过期时间
		return parseExpirationFromRawData(domain, whoisData)
	}

	// 解析过期时间
	slog.Debug("尝试解析过期时间", "domain", domain, "expiration_date", parsed.Domain.ExpirationDate)
	
	expiryDate, err := time.Parse("2006-01-02T15:04:05Z", parsed.Domain.ExpirationDate)
	if err != nil {
		// 尝试其他时间格式
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			"02-Jan-2006",
			"2006/01/02",
			"2006-01-02T15:04:05.000Z",
			"Mon Jan 02 15:04:05 MST 2006",
			"January 02 2006",
		}
		
		for _, format := range formats {
			if expiryDate, err = time.Parse(format, parsed.Domain.ExpirationDate); err == nil {
				slog.Debug("成功解析过期时间", "domain", domain, "format", format, "date", expiryDate)
				break
			}
		}
		
		if err != nil {
			slog.Error("无法解析过期时间", "domain", domain, "expiration_date", parsed.Domain.ExpirationDate, "error", err)
			// 尝试从原始数据中手动提取
			return parseExpirationFromRawData(domain, whoisData)
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

// GetDomainInfoWithFallback 使用WHOIS获取域名信息（带重试）
func GetDomainInfoWithFallback(domain string, timeout time.Duration, config *Config) (*DomainInfo, error) {
	maxRetries := 2
	var lastErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Debug("WHOIS查询尝试", "domain", domain, "attempt", attempt, "max_retries", maxRetries)
		
		info, err := GetDomainInfo(domain, timeout)
		if err == nil {
			if attempt > 1 {
				slog.Info("WHOIS查询重试成功", "domain", domain, "attempt", attempt)
			}
			return info, nil
		}
		
		lastErr = err
		slog.Debug("WHOIS查询失败", "domain", domain, "attempt", attempt, "error", err)
		
		// 如果不是最后一次尝试，等待一下再重试
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * time.Second
			slog.Debug("等待重试", "domain", domain, "wait_seconds", waitTime.Seconds())
			time.Sleep(waitTime)
		}
	}
	
	slog.Error("所有WHOIS查询尝试都失败了", "domain", domain, "attempts", maxRetries, "last_error", lastErr)
	return nil, fmt.Errorf("WHOIS查询失败: %v", lastErr)
}

// parseExpirationFromRawData 从原始WHOIS数据中手动提取过期时间
func parseExpirationFromRawData(domain, whoisData string) (*DomainInfo, error) {
	slog.Debug("尝试从原始数据手动解析过期时间", "domain", domain)
	
	// 常见的过期时间字段名
	expirationPatterns := []string{
		`(?i)Registry Expiry Date:\s*(.+)`,
		`(?i)Registrar Registration Expiration Date:\s*(.+)`,
		`(?i)Expiry Date:\s*(.+)`,
		`(?i)Expiration Date:\s*(.+)`,
		`(?i)Expires:\s*(.+)`,
		`(?i)Expire:\s*(.+)`,
		`(?i)Expiration Time:\s*(.+)`,
		`(?i)Registry Expiration Date:\s*(.+)`,
		`(?i)Domain Expiration Date:\s*(.+)`,
		`(?i)Paid-till:\s*(.+)`,
	}
	
	// 常见的注册商字段名
	registrarPatterns := []string{
		`(?i)Registrar:\s*(.+)`,
		`(?i)Sponsoring Registrar:\s*(.+)`,
		`(?i)Registrar Name:\s*(.+)`,
	}
	
	var expiryDate time.Time
	var registrar string
	var found bool
	
	// 尝试提取过期时间
	for _, pattern := range expirationPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(whoisData)
		if len(matches) > 1 {
			dateStr := strings.TrimSpace(matches[1])
			slog.Debug("找到过期时间字段", "domain", domain, "pattern", pattern, "date_str", dateStr)
			
			// 尝试解析日期
			if parsedDate, err := parseFlexibleDate(dateStr); err == nil {
				expiryDate = parsedDate
				found = true
				slog.Debug("成功解析过期时间", "domain", domain, "date", expiryDate)
				break
			} else {
				slog.Debug("解析日期失败", "domain", domain, "date_str", dateStr, "error", err)
			}
		}
	}
	
	if !found {
		return nil, fmt.Errorf("无法从原始数据中提取过期时间")
	}
	
	// 尝试提取注册商
	for _, pattern := range registrarPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(whoisData)
		if len(matches) > 1 {
			registrar = strings.TrimSpace(matches[1])
			break
		}
	}
	
	if registrar == "" {
		registrar = "Unknown"
	}
	
	return &DomainInfo{
		Domain:     domain,
		ExpiryDate: expiryDate,
		Registrar:  registrar,
		Status:     "active",
		Method:     "whois(manual_parse)",
	}, nil
}

// parseFlexibleDate 灵活解析各种日期格式
func parseFlexibleDate(dateStr string) (time.Time, error) {
	// 清理日期字符串
	dateStr = strings.TrimSpace(dateStr)
	
	// 移除常见的后缀
	dateStr = regexp.MustCompile(`\s+UTC`).ReplaceAllString(dateStr, "")
	dateStr = regexp.MustCompile(`\s+GMT`).ReplaceAllString(dateStr, "")
	dateStr = regexp.MustCompile(`\s+\+\d{4}`).ReplaceAllString(dateStr, "")
	
	// 尝试各种日期格式
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02-Jan-2006",
		"2006/01/02",
		"01/02/2006",
		"2006.01.02",
		"January 02 2006",
		"Jan 02 2006",
		"02 Jan 2006",
		"2006-01-02 15:04:05 UTC",
		"Mon Jan 02 15:04:05 MST 2006",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05+00:00",
	}
	
	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("无法解析日期格式: %s", dateStr)
}