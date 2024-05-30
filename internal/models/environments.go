package models

import (
	"database/sql"
	"errors"
	"github.com/vladComan0/tasty-byte/pkg/transactions"
	"golang.org/x/crypto/bcrypt"
	"log"
	"sort"
	"time"
)

const COST = 12 // 2^12 bcrypt iterations used to generate the password hash (4-31)

type EnvironmentModelInterface interface {
	Ping() error
	Insert(environment *Environment) (int, error)
	Get(id int) (*Environment, error)
	GetWithTx(tx transactions.Transaction, id int) (*Environment, error)
	GetAll() ([]*Environment, error)
	Update(environment *Environment) error
	Delete(id int) error
}

type Environment struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Endpoint      string    `json:"endpoint"`
	TokenEndpoint string    `json:"token_endpoint,omitempty"`
	Username      string    `json:"username,omitempty"`
	Password      string    `json:"password,omitempty"`
	Disabled      bool      `json:"disabled,omitempty"`
	CreatedAt     time.Time `json:"-"`
}

type EnvironmentModel struct {
	DB *sql.DB
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

func (m *EnvironmentModel) Ping() error {
	return m.DB.Ping()
}

func (m *EnvironmentModel) Insert(environment *Environment) (int, error) {
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
			(name, endpoint, token_endpoint, username, password, disabled, created_at)
		VALUES 
			(?, ?, ?, ?, ?, ?, UTC_TIMESTAMP())
		`
		result, err := tx.Exec(stmt, environment.Name, environment.Endpoint, environment.TokenEndpoint, environment.Username, hashedPassword, environment.Disabled)
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

func (m *EnvironmentModel) GetAll() ([]*Environment, error) {
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
			return nil, ErrNoRecord
		default:
			return nil, err
		}
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		var (
			environment = &Environment{}
		)

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

func (m *EnvironmentModel) GetWithTx(tx transactions.Transaction, id int) (*Environment, error) {
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
			return nil, ErrNoRecord
		default:
			return nil, err
		}
	}

	return environment, nil
}

func (m *EnvironmentModel) Get(id int) (*Environment, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			log.Printf("could not rollback %v", err)
		}
	}()

	return m.GetWithTx(tx, id)
}

func (m *EnvironmentModel) Update(environment *Environment) error {
	return transactions.WithTransaction(m.DB, func(tx transactions.Transaction) error {
		existingEnvironment, err := m.GetWithTx(tx, environment.ID)
		if err != nil {
			return err
		}

		if existingEnvironment == nil {
			return ErrNoRecord
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
			environment.Disabled,
			environment.ID,
		)
		if err != nil {
			return err
		}

		return nil
	})
}

func (m *EnvironmentModel) Delete(id int) error {
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
			return ErrNoRecord
		}

		return nil
	})
}
