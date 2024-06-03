package service

import (
	"github.com/rs/zerolog"
	"github.com/vladComan0/performance-analyzer/internal/custom_errors"
	"github.com/vladComan0/performance-analyzer/internal/data"
	"github.com/vladComan0/performance-analyzer/pkg/tokens"
	"sync"
)

type WorkerService interface {
	CreateWorker(input *data.Worker) (*data.Worker, error)
	GetWorker(id int) (*data.Worker, error)
	GetWorkers() ([]*data.Worker, error)
}

type WorkerServiceImpl struct {
	workerRepo      data.WorkerRepository
	environmentRepo data.EnvironmentRepository
	log             zerolog.Logger
}

func NewWorkerService(workerRepo data.WorkerRepository, environmentRepo data.EnvironmentRepository, log zerolog.Logger) *WorkerServiceImpl {
	return &WorkerServiceImpl{
		workerRepo:      workerRepo,
		environmentRepo: environmentRepo,
		log:             log,
	}
}

func (s *WorkerServiceImpl) CreateWorker(input *data.Worker) (*data.Worker, error) {
	if err := s.validateWorkerInput(input); err != nil {
		return nil, err
	}

	environment, err := s.environmentRepo.Get(input.EnvironmentID)
	if err != nil {
		return nil, err
	}

	if environment.Disabled {
		return nil, custom_errors.ErrEnvironmentDisabled
	}

	var options []data.WorkerOption

	if environment.TokenEndpoint != "" {
		credentials := tokens.Credentials{
			BasicAuthToken: &environment.BasicAuthToken,
		}
		tokenManager := tokens.NewTokenManager(credentials, environment.TokenEndpoint)
		options = append(options, data.WithWorkerTokenManager(tokenManager))
	}

	worker := data.NewWorker(
		input.EnvironmentID,
		input.Concurrency,
		input.RequestsPerTask,
		input.HTTPMethod,
		input.Body,
		environment,
		s.log,
		options...,
	)

	id, err := s.workerRepo.Insert(worker)
	if err != nil {
		return nil, err
	}

	// Fetch the worker details from the database using a dummy worker
	workerFromDB, err := s.workerRepo.Get(id)
	if err != nil {
		return nil, err
	}

	// Update the original worker with the relevant fields
	worker.ID = workerFromDB.ID
	worker.Status = workerFromDB.Status
	worker.CreatedAt = workerFromDB.CreatedAt

	wg := &sync.WaitGroup{}
	go worker.Start(wg, s.workerRepo.UpdateStatus, s.workerRepo.UpdateMetrics)

	return worker, nil
}

func (s *WorkerServiceImpl) GetWorker(id int) (*data.Worker, error) {
	return s.workerRepo.Get(id)
}

func (s *WorkerServiceImpl) GetWorkers() ([]*data.Worker, error) {
	return s.workerRepo.GetAll()
}

func (s *WorkerServiceImpl) validateWorkerInput(input *data.Worker) error {
	if input.EnvironmentID < 1 || input.Concurrency < 1 || input.RequestsPerTask < 1 {
		return custom_errors.ErrInvalidInput
	}
	return nil
}
