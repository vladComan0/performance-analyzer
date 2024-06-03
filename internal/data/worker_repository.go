package data

import (
	"database/sql"
	"errors"
	"github.com/vladComan0/performance-analyzer/internal/custom_errors"
	"github.com/vladComan0/tasty-byte/pkg/transactions"
	"sort"
)

type WorkerRepository interface {
	Insert(worker *Worker) (int, error)
	Get(id int) (*Worker, error)
	GetAll() ([]*Worker, error)
	UpdateStatus(id int, status Status) error
	UpdateMetrics(id int, metrics *Metrics) error
}

type WorkerRepositoryDB struct {
	DB *sql.DB
}

func NewWorkerRepositoryDB(db *sql.DB) *WorkerRepositoryDB {
	return &WorkerRepositoryDB{
		DB: db,
	}
}

func (m *WorkerRepositoryDB) Insert(worker *Worker) (int, error) {
	var workerID int

	err := transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		stmt := `
		INSERT INTO workers (environment_id, concurrency, requests_per_task, report, http_method, body, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP())
		`
		result, err := tx.Exec(
			stmt,
			worker.EnvironmentID,
			worker.Concurrency,
			worker.RequestsPerTask,
			worker.Report,
			worker.HTTPMethod,
			worker.Body,
			StatusCreated,
		)
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

func (m *WorkerRepositoryDB) GetAll() ([]*Worker, error) {
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
		body,
		status,
		max_latency,
		total_requests,
		failed_requests,
		error_rate,
		p50,
		p95,
		p99,
		p999,
		created_at
	FROM 
	    workers
	`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, custom_errors.ErrNoRecord
		default:
			return nil, err
		}
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		var worker = &Worker{}
		var p50, p95, p99, p999, maxLatency, errorRate sql.NullFloat64
		var totalRequests, failedRequests sql.NullInt64
		worker.Metrics = &Metrics{}
		worker.Metrics.Percentiles = make(map[PercentileRank]float64)

		err := rows.Scan(
			&worker.ID,
			&worker.EnvironmentID,
			&worker.Concurrency,
			&worker.RequestsPerTask,
			&worker.Report,
			&worker.HTTPMethod,
			&worker.Body,
			&worker.Status,
			&maxLatency,
			&totalRequests,
			&failedRequests,
			&errorRate,
			&p50,
			&p95,
			&p99,
			&p999,
			&worker.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		assignValidMetricsFromDB(worker, maxLatency, totalRequests, failedRequests, errorRate, p50, p95, p99, p999)

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

func (m *WorkerRepositoryDB) Get(id int) (*Worker, error) {
	var worker *Worker

	err := transactions.WithTransaction(m.DB, func(tx transactions.Transaction) (err error) {
		worker, err = m.getWithTx(tx, id)
		return err
	})

	return worker, err
}

func (m *WorkerRepositoryDB) getWithTx(tx transactions.Transaction, id int) (*Worker, error) {
	worker := &Worker{}
	worker.Metrics = &Metrics{}
	worker.Metrics.Percentiles = make(map[PercentileRank]float64)

	var p50, p95, p99, p999, maxLatency, errorRate sql.NullFloat64
	var totalRequests, failedRequests sql.NullInt64

	stmt := `
	SELECT
		id,
		environment_id,
		concurrency,
		requests_per_task,
		report,
		http_method,
		body,
		status,
		max_latency,
		total_requests,
		failed_requests,
		error_rate,
		p50,
		p95,
		p99,
		p999,
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
		&worker.Body,
		&worker.Status,
		&maxLatency,
		&totalRequests,
		&failedRequests,
		&errorRate,
		&p50,
		&p95,
		&p99,
		&p999,
		&worker.CreatedAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, custom_errors.ErrNoRecord
		default:
			return nil, err
		}
	}

	assignValidMetricsFromDB(worker, maxLatency, totalRequests, failedRequests, errorRate, p50, p95, p99, p999)

	return worker, nil
}

func (m *WorkerRepositoryDB) UpdateStatus(id int, newStatus Status) error {
	err := transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		stmt := `
		UPDATE workers
		SET status = ?
		WHERE id = ?
		`

		_, err := tx.Exec(stmt, newStatus, id)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (m *WorkerRepositoryDB) UpdateMetrics(id int, metrics *Metrics) error {
	err := transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		stmt := `
        UPDATE workers
        SET max_latency = ?,
            total_requests = ?,
            failed_requests = ?,
            error_rate = ?,
            p50 = ?,
            p95 = ?,
            p99 = ?,
            p999 = ?
        WHERE id = ?
        `

		_, err := tx.Exec(
			stmt,
			metrics.MaxLatency,
			metrics.TotalRequests,
			metrics.FailedRequests,
			metrics.ErrorRate,
			metrics.Percentiles[P50],
			metrics.Percentiles[P95],
			metrics.Percentiles[P99],
			metrics.Percentiles[P999],
			id,
		)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func assignValidMetricsFromDB(worker *Worker, maxLatency sql.NullFloat64, totalRequests, failedRequests sql.NullInt64, errorRate sql.NullFloat64, p50, p95, p99, p999 sql.NullFloat64) {
	if maxLatency.Valid {
		worker.Metrics.MaxLatency = maxLatency.Float64
	}

	if totalRequests.Valid {
		worker.Metrics.TotalRequests = int(totalRequests.Int64)
	}

	if failedRequests.Valid {
		worker.Metrics.FailedRequests = int(failedRequests.Int64)
	}

	if errorRate.Valid {
		worker.Metrics.ErrorRate = errorRate.Float64
	}

	if p50.Valid {
		worker.Metrics.Percentiles[P50] = p50.Float64
	}

	if p95.Valid {
		worker.Metrics.Percentiles[P95] = p95.Float64
	}

	if p99.Valid {
		worker.Metrics.Percentiles[P99] = p99.Float64
	}

	if p999.Valid {
		worker.Metrics.Percentiles[P999] = p999.Float64
	}
}
