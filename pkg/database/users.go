package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/sguter90/weathermaestro/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// hashPassword creates a SHA-256 hash of the password to handle passwords longer than 72 bytes
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// CreateUser creates a new user with hashed password
func (dm *DatabaseManager) CreateUser(ctx context.Context, username, password string) (*models.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("username and password must not be empty")
	}

	// Pre-hash password with SHA-256 to handle passwords longer than 72 bytes
	preHashedPassword := hashPassword(password)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(preHashedPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Mark as new format with prefix
	finalHash := "v2:" + string(hashedPassword)

	query := `
        INSERT INTO users (username, password_hash)
        VALUES ($1, $2)
        RETURNING id, username, created_at
    `

	var user models.User
	err = dm.QueryRowWithHealthCheck(ctx, query, username, finalHash).
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

	// Check if this is a new format hash (v2) or old format
	isNewFormat := strings.HasPrefix(passwordHash, "v2:")

	var compareErr error
	if isNewFormat {
		// New format: remove prefix and use SHA-256 pre-hashing
		actualHash := strings.TrimPrefix(passwordHash, "v2:")
		preHashedPassword := hashPassword(password)
		compareErr = bcrypt.CompareHashAndPassword([]byte(actualHash), []byte(preHashedPassword))
	} else {
		// Old format: direct bcrypt comparison
		compareErr = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))

		// If password is correct, migrate to new format
		if compareErr == nil {
			if err := dm.migrateUserPassword(ctx, user.ID, password); err != nil {
				// Log error but don't fail login
				fmt.Printf("Warning: failed to migrate password for user %s: %v\n", user.ID, err)
			}
		}
	}

	if compareErr != nil {
		return nil, errors.New("invalid credentials")
	}

	return &user, nil
}

// migrateUserPassword updates a user's password to the new format
func (dm *DatabaseManager) migrateUserPassword(ctx context.Context, userID interface{}, password string) error {
	preHashedPassword := hashPassword(password)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(preHashedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	finalHash := "v2:" + string(hashedPassword)

	query := `UPDATE users SET password_hash = $1 WHERE id = $2`
	_, err = dm.ExecWithHealthCheck(ctx, query, finalHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
