package data

import (
	"encoding/json"
	"github.com/rs/zerolog"
	"github.com/vladComan0/performance-analyzer/pkg/tokens"
	"net/http"
	"sync"
	"time"
)

type Worker struct {
	ID              int                  `json:"id"`
	EnvironmentID   int                  `json:"environment_id"`
	Concurrency     int                  `json:"concurrency"`
	RequestsPerTask int                  `json:"requests_per_task"`
	Report          string               `json:"report"`
	HTTPMethod      string               `json:"http_method"`
	Body            *json.RawMessage     `json:"body"`
	Status          Status               `json:"status"`
	CreatedAt       time.Time            `json:"-"`
	Metrics         *Metrics             `json:"-"`
	Environment     *Environment         `json:"-"`
	TokenManager    *tokens.TokenManager `json:"-"`
	log             zerolog.Logger
	mu              sync.Mutex
}

// NewWorker creates a new Worker with the given options.
func NewWorker(environmentID, concurrency, requestsPerTask int, httpMethod string, body *json.RawMessage, environment *Environment, log zerolog.Logger, options ...WorkerOption) *Worker {
	worker := &Worker{
		EnvironmentID:   environmentID,
		Concurrency:     concurrency,
		RequestsPerTask: requestsPerTask,
		Environment:     environment,
		HTTPMethod:      httpMethod,
		Body:            body,
		Status:          StatusCreated,
		Metrics:         NewMetrics(),
		log:             log,
	}

	for _, option := range options {
		option(worker)
	}

	return worker
}

func (w *Worker) Start(wg *sync.WaitGroup, updateFunc func(id int, status Status) error) {
	if err := updateFunc(w.ID, StatusRunning); err != nil {
		w.log.Error().Err(err).Msg("Error updating status to running")
		return
	}
	w.SetStatus(StatusRunning)

	for i := 0; i < w.Concurrency; i++ {
		wg.Add(1)
		go w.run(wg)
	}
	wg.Wait()

	if err := updateFunc(w.ID, StatusFinished); err != nil {
		w.log.Error().Err(err).Msg("Error updating status to finished")
		return
	}
	w.SetStatus(StatusFinished)

	ranks := []float64{50, 95, 99, 99.9}
	if err := w.Metrics.CalculatePercentiles(ranks...); err != nil {
		w.log.Error().Err(err).Msg("Error calculating Percentiles")
		return
	}

	w.log.Info().Msgf("p50 latency: %.6f s", w.Metrics.Percentiles[50]/1e9)
	w.log.Info().Msgf("p95 latency: %.6f s", w.Metrics.Percentiles[95]/1e9)
	w.log.Info().Msgf("p99 latency: %.6f s", w.Metrics.Percentiles[99]/1e9)
	w.log.Info().Msgf("p999 latency: %.6f s", w.Metrics.Percentiles[99.9]/1e9)

	w.Metrics.CalculateMaxLatency()

	w.log.Info().Msgf("Max latency: %.6f s", float64(w.Metrics.MaxLatency)/1e9)
	w.log.Info().Msgf("Total requests: %d", w.Metrics.TotalRequests)
	w.log.Info().Msgf("Failed requests: %d", w.Metrics.FailedRequests)
	w.log.Info().Msgf("Error rate: %.2f%%", 100*w.Metrics.CalculateErrorRate())
}

func (w *Worker) run(wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < w.RequestsPerTask; i++ {
		switch w.HTTPMethod {
		case http.MethodGet:
			w.get(w.Environment.Endpoint)
		case http.MethodPost:
			w.post(w.Environment.Endpoint)
		}
	}
}

func (w *Worker) get(url string) {
	client := &http.Client{}
	req, err := w.createRequest("GET", url)
	if err != nil {
		w.log.Error().Err(err).Msgf("Error creating request with HTTP method %s on the URL %s", w.HTTPMethod, url)
		return
	}

	w.log.Debug().Msgf("Sending request to: %s", url)

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	w.Metrics.IncrementTotalRequests()

	if err != nil {
		w.log.Error().Err(err).Msgf("Error sending request with HTTP method %s on the URL %s", w.HTTPMethod, url)
		w.Metrics.IncrementFailedRequests()
		return
	}
	defer resp.Body.Close()

	w.log.Debug().Msgf("Response status code: %s", resp.Status)

	w.Metrics.AddLatency(latency)
}

func (w *Worker) post(url string) {
	client := &http.Client{}
	req, err := w.createRequest("GET", url)
	if err != nil {
		w.log.Error().Err(err).Msgf("Error creating request with HTTP method %s on the URL %s", w.HTTPMethod, url)
		return
	}

	w.log.Debug().Msgf("Sending request to: %s", url)

	resp, err := client.Do(req)
	if err != nil {
		w.log.Error().Err(err).Msgf("Error sending request with HTTP method %s on the URL %s", w.HTTPMethod, url)
		return
	}
	defer resp.Body.Close()

	w.log.Debug().Msgf("Response status code: %s", resp.Status)
}

func (w *Worker) createRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if w.TokenManager != nil {
		token, err := w.TokenManager.GetToken()
		if err != nil {
			w.log.Error().Err(err).Msgf("Error fetching token on the URL %s", w.Environment.TokenEndpoint)
			return nil, err
		}
		w.log.Debug().Msgf("Token: %s", token)
		req.Header.Add("Authorization", "Bearer "+token)
	}

	req.Header.Add("Content-Type", "application/json")
	return req, nil
}
