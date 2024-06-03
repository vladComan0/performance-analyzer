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
		result, err := tx.Exec(stmt, worker.EnvironmentID, worker.Concurrency, worker.RequestsPerTask, worker.Report, worker.HTTPMethod, worker.Body, StatusCreated)
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
		created_at
	FROM workers
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

		err := rows.Scan(
			&worker.ID,
			&worker.EnvironmentID,
			&worker.Concurrency,
			&worker.RequestsPerTask,
			&worker.Report,
			&worker.HTTPMethod,
			&worker.Body,
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
