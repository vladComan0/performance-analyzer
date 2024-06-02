package data

import (
	"github.com/montanaflynn/stats"
	"sync"
	"time"
)

type Metrics struct {
	MaxLatency     time.Duration       `json:"max_latency,omitempty"`
	Percentiles    map[float64]float64 `json:"Percentiles,omitempty"`
	TotalRequests  int                 `json:"total_requests,omitempty"`
	FailedRequests int                 `json:"failed_requests,omitempty"`
	latencies      []time.Duration
	mu             sync.Mutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		Percentiles: make(map[float64]float64),
	}
}

func (m *Metrics) AddLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencies = append(m.latencies, latency)
}

func (m *Metrics) CalculateMaxLatency() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var maximum time.Duration
	for _, latency := range m.latencies {
		maximum = max(maximum, latency)
	}
	m.MaxLatency = maximum
}

func (m *Metrics) CalculatePercentiles(percentileRanks ...float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	latencies := make([]float64, len(m.latencies))
	for i, latency := range m.latencies {
		latencies[i] = float64(latency)
	}

	for _, rank := range percentileRanks {
		result, err := calculatePercentile(latencies, rank)
		if err != nil {
			return err
		}
		m.Percentiles[rank] = result
	}

	return nil
}

func calculatePercentile(latencies []float64, rank float64) (float64, error) {
	result, err := stats.Percentile(latencies, rank)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (m *Metrics) IncrementTotalRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalRequests++
}

func (m *Metrics) IncrementFailedRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedRequests++
}

func (m *Metrics) CalculateErrorRate() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TotalRequests == 0 {
		return 0
	}

	return float64(m.FailedRequests) / float64(m.TotalRequests)
}
