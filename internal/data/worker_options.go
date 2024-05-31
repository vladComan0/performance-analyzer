package data

import "github.com/vladComan0/performance-analyzer/pkg/tokens"

type WorkerOption func(*Worker)

func WithWorkerTokenManager(tokenManager *tokens.TokenManager) WorkerOption {
	return func(worker *Worker) {
		worker.TokenManager = tokenManager
	}
}

func WithWorkerReport(report string) WorkerOption {
	return func(worker *Worker) {
		worker.Report = report
	}
}
