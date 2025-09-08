package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"kevin197011.github.io/domain-exporter/checker"
)

// Metrics Prometheus metrics
type Metrics struct {
	domainExpiryDays *prometheus.GaugeVec
	domainValid      *prometheus.GaugeVec
	domainLastCheck  *prometheus.GaugeVec
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		domainExpiryDays: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "domain_expiry_days",
				Help: "Days remaining until domain registration expires",
			},
			[]string{"domain", "description"},
		),
		domainValid: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "domain_valid",
				Help: "Whether domain registration is valid (1=valid, 0=invalid)",
			},
			[]string{"domain", "description", "error"},
		),
		domainLastCheck: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "domain_last_check_timestamp",
				Help: "Timestamp of last domain registration check",
			},
			[]string{"domain", "description"},
		),
	}
}

// Register registers metrics to Prometheus
func (m *Metrics) Register() {
	prometheus.MustRegister(m.domainExpiryDays)
	prometheus.MustRegister(m.domainValid)
	prometheus.MustRegister(m.domainLastCheck)
}

// UpdateMetrics updates metrics data
func (m *Metrics) UpdateMetrics(domainInfos map[string]*checker.DomainInfo) {
	// Clear old metrics
	m.domainExpiryDays.Reset()
	m.domainValid.Reset()
	m.domainLastCheck.Reset()

	for _, info := range domainInfos {
		labels := prometheus.Labels{
			"domain":      info.Name,
			"description": info.Description,
		}

		// Update last check time
		m.domainLastCheck.With(labels).Set(float64(info.LastCheck.Unix()))

		if info.IsValid {
			// Domain is valid
			m.domainValid.With(prometheus.Labels{
				"domain":      info.Name,
				"description": info.Description,
				"error":       "",
			}).Set(1)

			// Days remaining
			m.domainExpiryDays.With(labels).Set(float64(info.DaysLeft))
		} else {
			// Domain is invalid
			m.domainValid.With(prometheus.Labels{
				"domain":      info.Name,
				"description": info.Description,
				"error":       info.Error,
			}).Set(0)

			// Set days remaining to -1 for invalid domains
			m.domainExpiryDays.With(labels).Set(-1)
		}
	}
}