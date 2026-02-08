package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sguter90/weathermaestro/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// CreateUser creates a new user with hashed password
func (dm *DatabaseManager) CreateUser(ctx context.Context, username, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
        INSERT INTO users (username, password_hash)
        VALUES ($1, $2)
        RETURNING id, username, created_at
    `

	var user models.User
	err = dm.QueryRowWithHealthCheck(ctx, query, username, string(hashedPassword)).
		Scan(&user.ID, &user.Username, &user.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// ValidateUser checks username and password
func (dm *DatabaseManager) ValidateUser(ctx context.Context, username, password string) (*models.User, error) {
	query := `
        SELECT id, username, password_hash, created_at
        FROM users
        WHERE username = $1
    `

	var user models.User
	var passwordHash string

	err := dm.QueryRowWithHealthCheck(ctx, query, username).
		Scan(&user.ID, &user.Username, &passwordHash, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid credentials")
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &user, nil
}
