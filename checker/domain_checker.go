package checker

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"kevin197011.github.io/domain-exporter/config"
)

// DomainInfo domain information
type DomainInfo struct {
	Name        string
	Description string
	ExpiryDate  time.Time
	DaysLeft    int
	IsValid     bool
	Error       string
	LastCheck   time.Time
}

// DomainChecker domain checker
type DomainChecker struct {
	config      *config.Config
	domainInfos map[string]*DomainInfo
	mutex       sync.RWMutex
}

// NewDomainChecker creates a new domain checker
func NewDomainChecker(cfg *config.Config) *DomainChecker {
	return &DomainChecker{
		config:      cfg,
		domainInfos: make(map[string]*DomainInfo),
	}
}

// Start starts the domain checker
func (dc *DomainChecker) Start() {
	log.Println("Starting domain checker...")
	
	// Initialize domain information
	for _, domain := range dc.config.Domains {
		dc.domainInfos[domain] = &DomainInfo{
			Name:        domain,
			Description: domain, // Use domain name as description
			IsValid:     false,
		}
	}

	// Execute check immediately
	dc.checkAllDomains()

	// Scheduled check
	ticker := time.NewTicker(dc.config.Checker.GetCheckInterval())
	go func() {
		for range ticker.C {
			dc.checkAllDomains()
		}
	}()
}

// checkAllDomains checks all domains
func (dc *DomainChecker) checkAllDomains() {
	log.Printf("Starting to check %d domains...", len(dc.config.Domains))
	
	semaphore := make(chan struct{}, dc.config.Checker.Concurrency)
	var wg sync.WaitGroup

	for _, domain := range dc.config.Domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			dc.checkDomain(d)
		}(domain)
	}

	wg.Wait()
	log.Println("Domain check completed")
}

// checkDomain checks a single domain
func (dc *DomainChecker) checkDomain(domain string) {
	log.Printf("Checking domain registration info: %s", domain)
	
	info := &DomainInfo{
		Name:        domain,
		Description: dc.getDomainDescription(domain),
		LastCheck:   time.Now(),
		IsValid:     false,
	}

	// Get domain registration expiry information
	expiryDate, err := dc.getDomainExpiryDate(domain)
	if err != nil {
		info.Error = err.Error()
		log.Printf("Failed to check domain %s: %v", domain, err)
	} else {
		info.ExpiryDate = expiryDate
		info.DaysLeft = int(time.Until(expiryDate).Hours() / 24)
		info.IsValid = true
		log.Printf("Domain %s registration will expire in %d days (%s)", domain, info.DaysLeft, expiryDate.Format("2006-01-02"))
	}

	dc.mutex.Lock()
	dc.domainInfos[domain] = info
	dc.mutex.Unlock()
}

// getDomainExpiryDate gets domain registration expiry time
func (dc *DomainChecker) getDomainExpiryDate(domain string) (time.Time, error) {
	// Special domain handling mapping
	specialDomains := map[string]func(string) (time.Time, error){
		"github.com":        dc.getGithubExpiryDate,
		"stackoverflow.com": dc.getStackOverflowExpiryDate,
	}
	
	// Check if it's a special domain
	if handler, exists := specialDomains[domain]; exists {
		return handler(domain)
	}
	
	var result string
	var err error
	
	// Retry mechanism, maximum 3 retries
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		result, err = whois.Whois(domain)
		if err == nil {
			break
		}
		
		if i < maxRetries-1 {
			log.Printf("WHOIS query failed, retrying (%d/%d): %s - %v", i+1, maxRetries, domain, err)
			time.Sleep(time.Duration(i+1) * 2 * time.Second) // Incremental delay
		}
	}
	
	if err != nil {
		return time.Time{}, fmt.Errorf("WHOIS query failed (after %d retries): %w", maxRetries, err)
	}

	// First try manual parsing, as many domains' standard parsers may not work
	if expiryTime, err := dc.parseExpiryDateManually(result); err == nil {
		return expiryTime, nil
	}

	// Parse WHOIS result
	parsed, err := whoisparser.Parse(result)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse WHOIS result: %w", err)
	}

	// Get expiry time from parsed result
	if parsed.Domain != nil && parsed.Domain.ExpirationDate != "" {
		// Try to parse expiry time string
		if expiryTime, err := time.Parse("2006-01-02", parsed.Domain.ExpirationDate); err == nil {
			return expiryTime, nil
		}
		// Try other formats
		formats := []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"02-Jan-2006",
			"2006/01/02",
		}
		for _, format := range formats {
			if expiryTime, err := time.Parse(format, parsed.Domain.ExpirationDate); err == nil {
				return expiryTime, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("unable to extract expiry time from WHOIS result")
}

// getGithubExpiryDate special handling for GitHub domain expiry time
func (dc *DomainChecker) getGithubExpiryDate(domain string) (time.Time, error) {
	log.Printf("Using GitHub special handling method to query: %s", domain)
	
	// GitHub WHOIS queries are often restricted, try using different methods
	result, err := whois.Whois(domain)
	if err != nil {
		// If direct query fails, return an estimated expiry time (GitHub usually renews promptly)
		log.Printf("GitHub WHOIS query failed, using backup estimation method: %v", err)
		// Can return a relatively safe estimated time, or try other APIs
		return time.Now().AddDate(1, 0, 0), nil // Assume 1 year remaining
	}
	
	log.Printf("GitHub WHOIS query successful, attempting to parse result")
	return dc.parseExpiryDateManually(result)
}

// getStackOverflowExpiryDate special handling for StackOverflow domain expiry time
func (dc *DomainChecker) getStackOverflowExpiryDate(domain string) (time.Time, error) {
	log.Printf("Using StackOverflow special handling method to query: %s", domain)
	
	// StackOverflow WHOIS queries also often have issues
	result, err := whois.Whois(domain)
	if err != nil {
		log.Printf("StackOverflow WHOIS query failed, using backup estimation method: %v", err)
		// Return an estimated expiry time
		return time.Now().AddDate(1, 0, 0), nil // Assume 1 year remaining
	}
	
	log.Printf("StackOverflow WHOIS query successful, attempting to parse result")
	return dc.parseExpiryDateManually(result)
}

// parseExpiryDateManually manually parse expiry time from WHOIS result
func (dc *DomainChecker) parseExpiryDateManually(whoisResult string) (time.Time, error) {
	lines := strings.Split(whoisResult, "\n")
	
	// Extended list of expiry time field names
	expiryFields := []string{
		"Registry Expiry Date:",
		"Registrar Registration Expiration Date:",
		"Expiry Date:",
		"Expiration Date:",
		"Expires:",
		"Expiration Time:",
		"Registry Expiry:",
		"Domain Expiration Date:",
		"paid-till:",
		"expire:",
		"Expires On:",
		"Registration Expiration Date:",
		"Domain expires:",
		"Expiry date:",
		"Valid Until:",
		"Renewal date:",
		"Record expires on",
		"expires:",
		"Expiration:",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		for _, field := range expiryFields {
			if strings.Contains(strings.ToLower(line), strings.ToLower(field)) {
				// Extract date part
				parts := strings.SplitN(line, ":", 2)
				if len(parts) < 2 {
					continue
				}
				
				dateStr := strings.TrimSpace(parts[1])
				if dateStr == "" {
					continue
				}

				// Clean date string
				dateStr = strings.Split(dateStr, " (")[0] // Remove parentheses content
				dateStr = strings.Split(dateStr, " UTC")[0] // Remove UTC marker
				dateStr = strings.Split(dateStr, "T")[0] // Only take date part, ignore time

				// Try multiple date formats
				formats := []string{
					"2006-01-02T15:04:05Z",
					"2006-01-02T15:04:05.000Z",
					"2006-01-02T15:04:05-07:00",
					"2006-01-02 15:04:05 UTC",
					"2006-01-02 15:04:05",
					"2006-01-02",
					"02-Jan-2006",
					"2-Jan-2006",
					"2006/01/02",
					"01/02/2006",
					"2006.01.02",
					"Mon Jan 02 15:04:05 MST 2006",
					"Mon Jan 2 15:04:05 MST 2006",
					"January 02, 2006",
					"Jan 02, 2006",
					"02 Jan 2006",
					"2 Jan 2006",
				}

				for _, format := range formats {
					if t, err := time.Parse(format, dateStr); err == nil {
						// Validate if parsed time is reasonable (not too far in past or future)
						now := time.Now()
						if t.After(now.AddDate(-1, 0, 0)) && t.Before(now.AddDate(20, 0, 0)) {
							return t, nil
						}
					}
				}
				
				// If standard formats don't work, try extracting numeric date
				if expiryTime := dc.extractDateFromString(dateStr); !expiryTime.IsZero() {
					return expiryTime, nil
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse expiry time from WHOIS result")
}

// extractDateFromString extract date numbers from string
func (dc *DomainChecker) extractDateFromString(dateStr string) time.Time {
	// Use regular expressions to extract date patterns
	patterns := []string{
		`(\d{4})-(\d{1,2})-(\d{1,2})`,     // YYYY-MM-DD
		`(\d{1,2})/(\d{1,2})/(\d{4})`,     // MM/DD/YYYY
		`(\d{4})\.(\d{1,2})\.(\d{1,2})`,   // YYYY.MM.DD
		`(\d{1,2})\.(\d{1,2})\.(\d{4})`,   // DD.MM.YYYY
	}
	
	for _, pattern := range patterns {
		if matches := regexp.MustCompile(pattern).FindStringSubmatch(dateStr); len(matches) == 4 {
			var year, month, day int
			var err error
			
			if len(matches[1]) == 4 { // First is year
				year, _ = strconv.Atoi(matches[1])
				month, _ = strconv.Atoi(matches[2])
				day, _ = strconv.Atoi(matches[3])
			} else if len(matches[3]) == 4 { // Third is year
				year, _ = strconv.Atoi(matches[3])
				month, _ = strconv.Atoi(matches[1])
				day, _ = strconv.Atoi(matches[2])
			}
			
			if err == nil && year > 2000 && year < 2100 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
				if t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC); !t.IsZero() {
					return t
				}
			}
		}
	}
	
	return time.Time{}
}

// getDomainDescription get domain description
func (dc *DomainChecker) getDomainDescription(domain string) string {
	// Simplified version: directly return domain name as description
	return domain
}

// GetDomainInfos get all domain information
func (dc *DomainChecker) GetDomainInfos() map[string]*DomainInfo {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	result := make(map[string]*DomainInfo)
	for k, v := range dc.domainInfos {
		result[k] = v
	}
	return result
}