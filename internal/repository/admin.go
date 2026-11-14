package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"tg-sender/internal/domain"
)

// AdminRepository handles admin data access.
type AdminRepository struct {
	db *pgxpool.Pool
}

// NewAdminRepository creates a new repository.
func NewAdminRepository(db *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{db: db}
}

// GetByUsername retrieves an admin by username.
func (r *AdminRepository) GetByUsername(ctx context.Context, username string) (*domain.Admin, error) {
	var a domain.Admin
	err := r.db.QueryRow(ctx, `
	    SELECT admin_id, username, password_hash, role, created_at
	    FROM admins WHERE username = $1`, username,
	).Scan(&a.AdminID, &a.Username, &a.PasswordHash, &a.Role, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get admin: %w", err)
	}
	return &a, nil
}

// Create inserts a new admin.
func (r *AdminRepository) Create(ctx context.Context, a *domain.Admin) error {
	_, err := r.db.Exec(ctx, `
	    INSERT INTO admins (admin_id, username, password_hash, role)
	    VALUES ($1, $2, $3, $4)`,
		a.AdminID, a.Username, a.PasswordHash, a.Role,
	)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	return nil
}
