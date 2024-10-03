package service

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/vladComan0/performance-analyzer/internal/custom_errors"
	"github.com/vladComan0/performance-analyzer/internal/model/entity"
	"github.com/vladComan0/performance-analyzer/internal/model/repository"
	"github.com/vladComan0/performance-analyzer/pkg/tokens"
	"sync"
)

type WorkerService interface {
	CreateWorker(ctx context.Context, input *entity.Worker) (*entity.Worker, error)
	GetWorker(id int) (*entity.Worker, error)
	GetWorkers() ([]*entity.Worker, error)
}

type WorkerServiceImpl struct {
	workerRepo      repository.WorkerRepository
	environmentRepo repository.EnvironmentRepository
	log             zerolog.Logger
}

func NewWorkerService(workerRepo repository.WorkerRepository, environmentRepo repository.EnvironmentRepository, log zerolog.Logger) *WorkerServiceImpl {
	return &WorkerServiceImpl{
		workerRepo:      workerRepo,
		environmentRepo: environmentRepo,
		log:             log,
	}
}

func (s *WorkerServiceImpl) CreateWorker(ctx context.Context, input *entity.Worker) (*entity.Worker, error) {
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

	var options []entity.WorkerOption

	if environment.TokenEndpoint != "" {
		credentials := tokens.Credentials{
			Username:       &environment.Username,
			Password:       &environment.Password,
			BasicAuthToken: &environment.BasicAuthToken,
		}
		tokenManager := tokens.NewTokenManager(credentials, environment.TokenEndpoint, s.log)
		options = append(options, entity.WithWorkerTokenManager(tokenManager))
	}

	worker := entity.NewWorker(
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
	go worker.Start(ctx, wg, s.workerRepo.UpdateStatus, s.workerRepo.UpdateMetrics)

	return worker, nil
}

func (s *WorkerServiceImpl) GetWorker(id int) (*entity.Worker, error) {
	return s.workerRepo.Get(id)
}

func (s *WorkerServiceImpl) GetWorkers() ([]*entity.Worker, error) {
	return s.workerRepo.GetAll()
}

func (s *WorkerServiceImpl) validateWorkerInput(input *entity.Worker) error {
	if input.EnvironmentID < 1 || input.Concurrency < 1 || input.RequestsPerTask < 1 {
		return custom_errors.ErrInvalidInput
	}
	return nil
}
