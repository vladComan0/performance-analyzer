package repository

import (
	"database/sql"
	"errors"
	"github.com/vladComan0/performance-analyzer/internal/custom_errors"
	"github.com/vladComan0/performance-analyzer/internal/model/entity"
	"github.com/vladComan0/tasty-byte/pkg/transactions"
	"golang.org/x/crypto/bcrypt"
	"sort"
)

const COST = 12 // 2^12 bcrypt iterations used to generate the password hash (4-31)

type EnvironmentRepository interface {
	Ping() error
	Insert(environment *entity.Environment) (int, error)
	Get(id int) (*entity.Environment, error)
	GetAll() ([]*entity.Environment, error)
	Update(environment *entity.Environment) error
	Delete(id int) error
}

type EnvironmentRepositoryDB struct {
	DB *sql.DB
}

func NewEnvironmentRepositoryDB(db *sql.DB) *EnvironmentRepositoryDB {
	return &EnvironmentRepositoryDB{
		DB: db,
	}
}

func (m *EnvironmentRepositoryDB) Ping() error {
	return m.DB.Ping()
}

func (m *EnvironmentRepositoryDB) Insert(environment *entity.Environment) (int, error) {
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

func (m *EnvironmentRepositoryDB) GetAll() ([]*entity.Environment, error) {
	var results []*entity.Environment
	environments := make(map[int]*entity.Environment)

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
		var environment = &entity.Environment{}

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

func (m *EnvironmentRepositoryDB) Get(id int) (*entity.Environment, error) {
	var environment *entity.Environment

	err := transactions.WithTransaction(m.DB, func(tx transactions.Transaction) (err error) {
		environment, err = m.getWithTx(tx, id)
		return err
	})

	return environment, err
}

func (m *EnvironmentRepositoryDB) Update(environment *entity.Environment) error {
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

func (m *EnvironmentRepositoryDB) Delete(id int) error {
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

func (m *EnvironmentRepositoryDB) getWithTx(tx transactions.Transaction, id int) (*entity.Environment, error) {
	environment := &entity.Environment{}

	stmt := `
    SELECT 
        id, 
        name, 
        endpoint,
        token_endpoint,
        username,
        password,
        basic_auth_token,
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
		&environment.Username,
		&environment.Password,
		&environment.BasicAuthToken,
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
