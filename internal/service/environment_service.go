package service

import (
	"github.com/vladComan0/performance-analyzer/internal/data"
	"github.com/vladComan0/performance-analyzer/internal/dto"
)

type EnvironmentService interface {
	PingDB() error
	CreateEnvironment(input dto.CreateEnvironmentInput) (*data.Environment, error)
	GetEnvironment(id int) (*data.Environment, error)
	GetEnvironments() ([]*data.Environment, error)
	UpdateEnvironment(id int, input dto.UpdateEnvironmentInput) (*data.Environment, error)
	DeleteEnvironment(id int) error
}

type EnvironmentServiceImpl struct {
	environmentRepo data.EnvironmentRepository
}

func NewEnvironmentService(environmentRepo data.EnvironmentRepository) *EnvironmentServiceImpl {
	return &EnvironmentServiceImpl{
		environmentRepo: environmentRepo,
	}
}

func (s *EnvironmentServiceImpl) PingDB() error {
	return s.environmentRepo.Ping()
}

func (s *EnvironmentServiceImpl) CreateEnvironment(input dto.CreateEnvironmentInput) (*data.Environment, error) {
	var options []data.EnvironmentOption
	if input.TokenEndpoint != nil {
		options = append(options, data.WithEnvironmentTokenEndpoint(*input.TokenEndpoint))
	}
	if input.Username != nil {
		options = append(options, data.WithEnvironmentUsername(*input.Username))
	}
	if input.Password != nil {
		options = append(options, data.WithEnvironmentPassword(*input.Password))
	}
	if input.Disabled != nil {
		options = append(options, data.WithEnvironmentDisabled(*input.Disabled))

	}

	environment := data.NewEnvironment(input.Name, input.Endpoint, options...)
	id, err := s.environmentRepo.Insert(environment)
	if err != nil {
		return nil, err
	}
	return s.environmentRepo.Get(id)
}

func (s *EnvironmentServiceImpl) GetEnvironment(id int) (*data.Environment, error) {
	return s.environmentRepo.Get(id)
}

func (s *EnvironmentServiceImpl) GetEnvironments() ([]*data.Environment, error) {
	return s.environmentRepo.GetAll()
}

func (s *EnvironmentServiceImpl) UpdateEnvironment(id int, input dto.UpdateEnvironmentInput) (*data.Environment, error) {
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
