package entity

import (
	"github.com/rs/zerolog"
	"sync"
	"testing"
	"time"
)

func BenchmarkChannelApproach(b *testing.B) {
	env := &Environment{
		ID:             8,
		Name:           "Meli",
		Disabled:       false,
		TokenEndpoint:  "http://localhost:8080/token",
		BasicAuthToken: "Basic YWR",
		Endpoint:       "https://youtube.com",
		CreatedAt:      time.Now(),
	}
	for i := 0; i < b.N; i++ {
		worker := NewWorker(5, 50, 100, "GET", nil, env, zerolog.Logger{})

		wg := &sync.WaitGroup{}
		// Start the worker
		worker.Start(wg, func(id int, status Status) error {
			return nil
		}, func(id int, metrics *Metrics) error {
			return nil
		})
	}
}

func BenchmarkTraditionalApproach(b *testing.B) {
	env := &Environment{
		ID:             8,
		Name:           "Meli",
		Disabled:       false,
		TokenEndpoint:  "http://localhost:8080/token",
		BasicAuthToken: "Basic YWR",
		Endpoint:       "https://youtube.com",
		CreatedAt:      time.Now(),
	}
	for i := 0; i < b.N; i++ {
		worker := NewWorker(5, 50, 100, "GET", nil, env, zerolog.Logger{})

		wg := &sync.WaitGroup{}
		// Start the worker
		worker.Start2(wg, func(id int, status Status) error {
			return nil
		}, func(id int, metrics *Metrics) error {
			return nil
		})
	}
}
