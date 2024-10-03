package service

import (
	"github.com/vladComan0/performance-analyzer/internal/dto"
	"github.com/vladComan0/performance-analyzer/internal/model/entity"
	"github.com/vladComan0/performance-analyzer/internal/model/repository"
)

type EnvironmentService interface {
	PingDB() error
	CreateEnvironment(input dto.CreateEnvironmentInput) (*entity.Environment, error)
	GetEnvironment(id int) (*entity.Environment, error)
	GetEnvironments() ([]*entity.Environment, error)
	UpdateEnvironment(id int, input dto.UpdateEnvironmentInput) (*entity.Environment, error)
	DeleteEnvironment(id int) error
}

type EnvironmentServiceImpl struct {
	environmentRepo repository.EnvironmentRepository
}

func NewEnvironmentService(environmentRepo repository.EnvironmentRepository) *EnvironmentServiceImpl {
	return &EnvironmentServiceImpl{
		environmentRepo: environmentRepo,
	}
}

func (s *EnvironmentServiceImpl) PingDB() error {
	return s.environmentRepo.Ping()
}

func (s *EnvironmentServiceImpl) CreateEnvironment(input dto.CreateEnvironmentInput) (*entity.Environment, error) {
	var options []entity.EnvironmentOption
	if input.TokenEndpoint != nil {
		options = append(options, entity.WithEnvironmentTokenEndpoint(*input.TokenEndpoint))
	}
	if input.Username != nil {
		options = append(options, entity.WithEnvironmentUsername(*input.Username))
	}
	if input.Password != nil {
		options = append(options, entity.WithEnvironmentPassword(*input.Password))
	}
	if input.Disabled != nil {
		options = append(options, entity.WithEnvironmentDisabled(*input.Disabled))

	}

	environment := entity.NewEnvironment(input.Name, input.Endpoint, options...)
	id, err := s.environmentRepo.Insert(environment)
	if err != nil {
		return nil, err
	}
	return s.environmentRepo.Get(id)
}

func (s *EnvironmentServiceImpl) GetEnvironment(id int) (*entity.Environment, error) {
	return s.environmentRepo.Get(id)
}

func (s *EnvironmentServiceImpl) GetEnvironments() ([]*entity.Environment, error) {
	return s.environmentRepo.GetAll()
}

func (s *EnvironmentServiceImpl) UpdateEnvironment(id int, input dto.UpdateEnvironmentInput) (*entity.Environment, error) {
	environment, err := s.environmentRepo.Get(id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		environment.Name = *input.Name
	}

	if input.Endpoint != nil {
		environment.Endpoint = *input.Endpoint
	}

	if input.TokenEndpoint != nil {
		environment.TokenEndpoint = *input.TokenEndpoint
	}

	if input.Username != nil {
		environment.Username = *input.Username
	}

	if input.Password != nil {
		environment.Password = *input.Password
	}

	if input.Disabled != nil {
		environment.Disabled = *input.Disabled
	}

	if err := s.environmentRepo.Update(environment); err != nil {
		return nil, err
	}

	return s.environmentRepo.Get(environment.ID) // to get the updated environment without the password
}

func (s *EnvironmentServiceImpl) DeleteEnvironment(id int) error {
	return s.environmentRepo.Delete(id)
}
