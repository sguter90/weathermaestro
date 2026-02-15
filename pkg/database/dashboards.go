package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// CreateDashboard creates a new dashboard
func (dm *DatabaseManager) CreateDashboard(ctx context.Context, dashboard *models.Dashboard) error {
	query := `
        INSERT INTO dashboards (name, description, config, is_default)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at
    `

	err := dm.QueryRowWithHealthCheck(ctx, query,
		dashboard.Name,
		dashboard.Description,
		dashboard.Config,
		dashboard.IsDefault,
	).Scan(&dashboard.ID, &dashboard.CreatedAt, &dashboard.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	return nil
}

// GetDashboards retrieves all dashboards
func (dm *DatabaseManager) GetDashboards(ctx context.Context) ([]models.Dashboard, error) {
	query := `
        SELECT id, name, description, config, is_default, created_at, updated_at
        FROM dashboards
        ORDER BY is_default DESC, created_at DESC
    `

	rows, err := dm.QueryWithHealthCheck(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query dashboards: %w", err)
	}
	defer rows.Close()

	dashboards := []models.Dashboard{}
	for rows.Next() {
		var d models.Dashboard
		err := rows.Scan(
			&d.ID,
			&d.Name,
			&d.Description,
			&d.Config,
			&d.IsDefault,
			&d.CreatedAt,
			&d.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dashboard: %w", err)
		}
		dashboards = append(dashboards, d)
	}

	return dashboards, nil
}

// GetDashboard retrieves a single dashboard by ID
func (dm *DatabaseManager) GetDashboard(ctx context.Context, id uuid.UUID) (*models.Dashboard, error) {
	query := `
        SELECT id, name, description, config, is_default, created_at, updated_at
        FROM dashboards
        WHERE id = $1
    `

	var d models.Dashboard
	err := dm.QueryRowWithHealthCheck(ctx, query, id).Scan(
		&d.ID,
		&d.Name,
		&d.Description,
		&d.Config,
		&d.IsDefault,
		&d.CreatedAt,
		&d.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dashboard not found")
		}
		return nil, fmt.Errorf("failed to query dashboard: %w", err)
	}

	return &d, nil
}

// UpdateDashboard updates an existing dashboard
func (dm *DatabaseManager) UpdateDashboard(ctx context.Context, dashboard *models.Dashboard) error {
	query := `
        UPDATE dashboards
        SET name = $1, description = $2, config = $3, is_default = $4, updated_at = $5
        WHERE id = $6
    `

	dashboard.UpdatedAt = time.Now()

	result, err := dm.ExecWithHealthCheck(ctx, query,
		dashboard.Name,
		dashboard.Description,
		dashboard.Config,
		dashboard.IsDefault,
		dashboard.UpdatedAt,
		dashboard.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("dashboard not found")
	}

	return nil
}

// DeleteDashboard deletes a dashboard
func (dm *DatabaseManager) DeleteDashboard(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM dashboards WHERE id = $1`

	result, err := dm.ExecWithHealthCheck(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("dashboard not found")
	}

	return nil
}

// GetDefaultDashboard retrieves the default dashboard
func (dm *DatabaseManager) GetDefaultDashboard(ctx context.Context) (*models.Dashboard, error) {
	query := `
        SELECT id, name, description, config, is_default, created_at, updated_at
        FROM dashboards
        WHERE is_default = true
        LIMIT 1
    `

	var d models.Dashboard
	err := dm.QueryRowWithHealthCheck(ctx, query).Scan(
		&d.ID,
		&d.Name,
		&d.Description,
		&d.Config,
		&d.IsDefault,
		&d.CreatedAt,
		&d.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No default dashboard
		}
		return nil, fmt.Errorf("failed to query default dashboard: %w", err)
	}

	return &d, nil
}
