package data

import (
	"database/sql"
	"errors"
	errors2 "github.com/vladComan0/performance-analyzer/internal/errors"
	"github.com/vladComan0/performance-analyzer/pkg/tokens"
	"github.com/vladComan0/tasty-byte/pkg/transactions"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type WorkerStorageInterface interface {
	Insert(worker *Worker) (int, error)
	Get(id int) (*Worker, error)
	GetAll() ([]*Worker, error)
}

type WorkerStorage struct {
	DB *sql.DB
}

type Worker struct {
	ID              int       `json:"id"`
	EnvironmentID   int       `json:"environment_id"`
	Concurrency     int       `json:"concurrency"`
	RequestsPerTask int       `json:"requests_per_task"`
	Report          string    `json:"report"`
	HTTPMethod      string    `json:"http_method"`
	Status          Status    `json:"status"`
	CreatedAt       time.Time `json:"-"`
	infoLog         *log.Logger
	errorLog        *log.Logger
	Environment     *Environment
	TokenManager    *tokens.TokenManager
}

// NewWorker creates a new Worker with the given options.
func NewWorker(environmentID, concurrency, requestsPerTask int, httpMethod string, environment *Environment, infoLog *log.Logger, errorLog *log.Logger, options ...WorkerOption) *Worker {
	worker := &Worker{
		EnvironmentID:   environmentID,
		Concurrency:     concurrency,
		RequestsPerTask: requestsPerTask,
		Environment:     environment,
		HTTPMethod:      httpMethod,
		Status:          StatusCreated,
		infoLog:         infoLog,
		errorLog:        errorLog,
	}

	for _, option := range options {
		option(worker)
	}

	return worker
}

func (w *Worker) Start(wg *sync.WaitGroup) {
	w.SetStatus(StatusRunning)
	for i := 0; i < w.Concurrency; i++ {
		wg.Add(1)
		go w.run(wg)
	}
	wg.Wait()
	w.SetStatus(StatusFinished)
}

func (w *Worker) run(wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < w.RequestsPerTask; i++ {
		switch w.HTTPMethod {
		case http.MethodGet:
			w.get(w.Environment.Endpoint)
		case http.MethodPost:
			// implement
		case http.MethodPut:
			// implement
		case http.MethodDelete:
			// implement
		}
	}
}

func (w *Worker) get(url string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		w.errorLog.Printf("Error creating request with HTTP method %s on the URL %s: %s", w.HTTPMethod, url, err)
		return
	}

	if w.TokenManager != nil {
		token, err := w.TokenManager.GetToken()
		if err != nil {
			w.errorLog.Printf("Error fetching token on the URL %s: %s ", w.Environment.TokenEndpoint, err)
			return
		}
		w.infoLog.Printf("Token: %s", token)
		req.Header.Add("Authorization", "Bearer "+token)
	}

	req.Header.Add("Content-Type", "application/json")

	w.infoLog.Printf("Sending request to: %s", url)

	resp, err := client.Do(req)
	if err != nil {
		w.errorLog.Printf("Error sending request with HTTP method %s to %s: %s", w.HTTPMethod, url, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Status code: %d", resp.StatusCode)
}

func (m *WorkerStorage) Insert(worker *Worker) (int, error) {
	var workerID int

	err := transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		stmt := `
		INSERT INTO workers (environment_id, concurrency, requests_per_task, report, http_method, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, UTC_TIMESTAMP())
		`
		result, err := tx.Exec(stmt, worker.EnvironmentID, worker.Concurrency, worker.RequestsPerTask, worker.Report, worker.HTTPMethod, worker.Status)
		if err != nil {
			return err
		}

		workerID64, err := result.LastInsertId()
		if err != nil {
			return err
		}
		workerID = int(workerID64)

		return nil
	})

	return workerID, err
}

func (m *WorkerStorage) GetAll() ([]*Worker, error) {
	var results []*Worker
	workers := make(map[int]*Worker)

	stmt := `
	SELECT
		id,
		environment_id,
		concurrency,
		requests_per_task,
		report,
		http_method,
		status,
		created_at
	FROM workers
	`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, errors2.ErrNoRecord
		default:
			return nil, err
		}
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		var worker = &Worker{}

		err := rows.Scan(
			&worker.ID,
			&worker.EnvironmentID,
			&worker.Concurrency,
			&worker.RequestsPerTask,
			&worker.Report,
			&worker.HTTPMethod,
			&worker.Status,
			&worker.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if _, exists := workers[worker.ID]; !exists {
			workers[worker.ID] = worker
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	for _, worker := range workers {
		results = append(results, worker)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return results, nil
}

func (m *WorkerStorage) Get(id int) (*Worker, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			log.Printf("could not rollback %v", err)
		}
	}()

	return m.getWithTx(tx, id)
}

func (m *WorkerStorage) getWithTx(tx transactions.Transaction, id int) (*Worker, error) {
	worker := &Worker{}

	stmt := `
	SELECT 
		id, 
		environment_id,
		concurrency,
		requests_per_task,
		report,
		http_method,
		status,
		created_at
	FROM 
	    workers
	WHERE id = ?
	`

	err := tx.QueryRow(stmt, id).Scan(
		&worker.ID,
		&worker.EnvironmentID,
		&worker.Concurrency,
		&worker.RequestsPerTask,
		&worker.Report,
		&worker.HTTPMethod,
		&worker.Status,
		&worker.CreatedAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, errors2.ErrNoRecord
		default:
			return nil, err
		}
	}

	return worker, nil
}
