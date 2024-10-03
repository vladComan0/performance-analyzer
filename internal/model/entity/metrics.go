package entity

import (
	"github.com/montanaflynn/stats"
	"strconv"
	"sync"
	"time"
)

type Metrics struct {
	MaxLatency     float64                    `json:"max_latency"` // in seconds
	Percentiles    map[PercentileRank]float64 `json:"percentiles"` // in seconds
	TotalRequests  int                        `json:"total_requests"`
	FailedRequests int                        `json:"failed_requests"`
	ErrorRate      float64                    `json:"error_rate"`
	latencies      []time.Duration
	mu             sync.Mutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		Percentiles: make(map[PercentileRank]float64),
	}
}

type PercentileRank string

const (
	P50  PercentileRank = "50"
	P95  PercentileRank = "95"
	P99  PercentileRank = "99"
	P999 PercentileRank = "99.9"
)

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
	m.MaxLatency = float64(maximum) / float64(time.Second)
}

func (m *Metrics) CalculatePercentiles(percentileRanks ...PercentileRank) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	latencies := make([]float64, len(m.latencies))
	for i, latency := range m.latencies {
		latencies[i] = float64(latency) / float64(time.Second)
	}

	for _, rank := range percentileRanks {
		rankFloat, err := strconv.ParseFloat(string(rank), 64)
		if err != nil {
			return err
		}
		result, err := calculatePercentile(latencies, rankFloat)
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

func (m *Metrics) CalculateErrorRate() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TotalRequests == 0 {
		m.ErrorRate = 0
		return
	}

	m.ErrorRate = float64(m.FailedRequests) / float64(m.TotalRequests)
}
