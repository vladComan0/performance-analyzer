package data

import (
	"database/sql"
	"errors"
	"github.com/vladComan0/performance-analyzer/internal/custom_errors"
	"github.com/vladComan0/tasty-byte/pkg/transactions"
	"golang.org/x/crypto/bcrypt"
	"log"
	"sort"
	"time"
)

const COST = 12 // 2^12 bcrypt iterations used to generate the password hash (4-31)

type EnvironmentStorageInterface interface {
	Ping() error
	Insert(environment *Environment) (int, error)
	Get(id int) (*Environment, error)
	GetAll() ([]*Environment, error)
	Update(environment *Environment) error
	Delete(id int) error
}

type EnvironmentStorage struct {
	DB *sql.DB
}

type Environment struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Endpoint       string    `json:"endpoint"`
	TokenEndpoint  string    `json:"token_endpoint,omitempty"`
	Username       string    `json:"username,omitempty"`
	Password       string    `json:"password,omitempty"`
	BasicAuthToken string    `json:"basic_auth_token,omitempty"`
	Disabled       bool      `json:"disabled,omitempty"`
	CreatedAt      time.Time `json:"-"`
}

// NewEnvironment creates a new Environment with the given options.
func NewEnvironment(name, endpoint string, options ...EnvironmentOption) *Environment {
	environment := &Environment{
		Name:     name,
		Endpoint: endpoint,
	}

	for _, opt := range options {
		opt(environment)
	}

	return environment
}

func (m *EnvironmentStorage) Ping() error {
	return m.DB.Ping()
}

func (m *EnvironmentStorage) Insert(environment *Environment) (int, error) {
	var (
		environmentID  int
		hashedPassword []byte
		err            error
	)

	if environment.Password != "" {
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(environment.Password), COST)
		if err != nil {
			return 0, err
		}
	}

	err = transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		stmt := `
		INSERT INTO environments 
			(name, endpoint, token_endpoint, username, password, basic_auth_token, disabled, created_at)
		VALUES 
			(?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP())
		`
		result, err := tx.Exec(stmt, environment.Name, environment.Endpoint, environment.TokenEndpoint, environment.Username, hashedPassword, environment.BasicAuthToken, environment.Disabled)
		if err != nil {
			return err
		}

		environmentID64, err := result.LastInsertId()
		if err != nil {
			return err
		}
		environmentID = int(environmentID64)

		return nil
	})

	return environmentID, err
}

func (m *EnvironmentStorage) GetAll() ([]*Environment, error) {
	var results []*Environment
	environments := make(map[int]*Environment)

	stmt := `
	SELECT 
		id,
		name,
		endpoint,
		token_endpoint,
		disabled,
		created_at
	FROM
		environments
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
		var environment = &Environment{}

		err := rows.Scan(
			&environment.ID,
			&environment.Name,
			&environment.Endpoint,
			&environment.TokenEndpoint,
			&environment.Disabled,
			&environment.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if _, exists := environments[environment.ID]; !exists {
			environments[environment.ID] = environment
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	for _, environment := range environments {
		results = append(results, environment)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return results, nil
}

func (m *EnvironmentStorage) Get(id int) (*Environment, error) {
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

func (m *EnvironmentStorage) Update(environment *Environment) error {
	return transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		existingEnvironment, err := m.getWithTx(tx, environment.ID)
		if err != nil {
			return err
		}

		if existingEnvironment == nil {
			return custom_errors.ErrNoRecord
		}

		hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(environment.Password), COST)
		if err != nil {
			return err
		}

		stmt := `
		UPDATE environments
		SET 
			name = ?, 
			endpoint = ?,
			token_endpoint = ?,
			username = ?,
			password = ?,
			basic_auth_token = ?,
			disabled = ?
		WHERE 
			id = ?
		`
		_, err = tx.Exec(
			stmt,
			environment.Name,
			environment.Endpoint,
			environment.TokenEndpoint,
			environment.Username,
			hashedNewPassword,
			environment.BasicAuthToken,
			environment.Disabled,
			environment.ID,
		)
		if err != nil {
			return err
		}

		return nil
	})
}

func (m *EnvironmentStorage) Delete(id int) error {
	return transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		stmt := `
		DELETE FROM environments
		WHERE id = ?
		`
		results, err := tx.Exec(stmt, id)
		if err != nil {
			return err
		}

		rowsAffected, err := results.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return custom_errors.ErrNoRecord
		}

		return nil
	})
}

func (m *EnvironmentStorage) getWithTx(tx transactions.Transaction, id int) (*Environment, error) {
	environment := &Environment{}

	stmt := `
    SELECT 
        id, 
        name, 
        endpoint,
        token_endpoint,
		disabled,
		created_at
    FROM 
        environments 
    WHERE 
        id = ?
	`

	err := tx.QueryRow(stmt, id).Scan(
		&environment.ID,
		&environment.Name,
		&environment.Endpoint,
		&environment.TokenEndpoint,
		&environment.Disabled,
		&environment.CreatedAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, custom_errors.ErrNoRecord
		default:
			return nil, err
		}
	}

	return environment, nil
}
