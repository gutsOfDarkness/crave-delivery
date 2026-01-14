// Package repository implements menu item data access
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"fooddelivery/internal/domain"
	"fooddelivery/pkg/database"
)

// MenuRepository handles menu item data persistence
type MenuRepository struct {
	db *database.Pool
}

// NewMenuRepository creates a new menu repository
func NewMenuRepository(db *database.Pool) *MenuRepository {
	return &MenuRepository{db: db}
}

// GetAll retrieves all available menu items
func (r *MenuRepository) GetAll(ctx context.Context) ([]domain.MenuItem, error) {
	query := `
		SELECT id, name, description, price, category, image_url, is_available, created_at, updated_at
		FROM menu_items
		WHERE is_available = TRUE
		ORDER BY category, name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query menu items: %w", err)
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		var imageURL *string

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Category,
			&imageURL,
			&item.IsAvailable,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan menu item: %w", err)
		}

		if imageURL != nil {
			item.ImageURL = *imageURL
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating menu items: %w", err)
	}

	return items, nil
}

// GetAllIncludingUnavailable retrieves all menu items (admin view)
func (r *MenuRepository) GetAllIncludingUnavailable(ctx context.Context) ([]domain.MenuItem, error) {
	query := `
		SELECT id, name, description, price, category, image_url, is_available, created_at, updated_at
		FROM menu_items
		ORDER BY category, name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query menu items: %w", err)
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		var imageURL *string

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Category,
			&imageURL,
			&item.IsAvailable,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan menu item: %w", err)
		}

		if imageURL != nil {
			item.ImageURL = *imageURL
		}

		items = append(items, item)
	}

	return items, nil
}

// GetByID retrieves a menu item by UUID
func (r *MenuRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error) {
	query := `
		SELECT id, name, description, price, category, image_url, is_available, created_at, updated_at
		FROM menu_items
		WHERE id = $1
	`

	item := &domain.MenuItem{}
	var imageURL *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.Category,
		&imageURL,
		&item.IsAvailable,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get menu item: %w", err)
	}

	if imageURL != nil {
		item.ImageURL = *imageURL
	}

	return item, nil
}

// GetByIDs retrieves multiple menu items by their UUIDs
// Used for order creation to validate and fetch prices server-side
func (r *MenuRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.MenuItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, name, description, price, category, image_url, is_available, created_at, updated_at
		FROM menu_items
		WHERE id = ANY($1) AND is_available = TRUE
	`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query menu items by IDs: %w", err)
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		var imageURL *string

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Category,
			&imageURL,
			&item.IsAvailable,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan menu item: %w", err)
		}

		if imageURL != nil {
			item.ImageURL = *imageURL
		}

		items = append(items, item)
	}

	return items, nil
}

// Create inserts a new menu item
func (r *MenuRepository) Create(ctx context.Context, item *domain.MenuItem) error {
	query := `
		INSERT INTO menu_items (id, name, description, price, category, image_url, is_available, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	item.ID = uuid.New()
	_, err := r.db.Exec(ctx, query,
		item.ID,
		item.Name,
		item.Description,
		item.Price,
		item.Category,
		item.ImageURL,
		item.IsAvailable,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create menu item: %w", err)
	}

	return nil
}

// Update modifies an existing menu item
func (r *MenuRepository) Update(ctx context.Context, item *domain.MenuItem) error {
	query := `
		UPDATE menu_items
		SET name = $2, description = $3, price = $4, category = $5, 
		    image_url = $6, is_available = $7, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		item.ID,
		item.Name,
		item.Description,
		item.Price,
		item.Category,
		item.ImageURL,
		item.IsAvailable,
	)

	if err != nil {
		return fmt.Errorf("failed to update menu item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes a menu item (soft delete by setting is_available = false)
func (r *MenuRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE menu_items
		SET is_available = FALSE, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete menu item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// GetByCategory retrieves menu items by category
func (r *MenuRepository) GetByCategory(ctx context.Context, category string) ([]domain.MenuItem, error) {
	query := `
		SELECT id, name, description, price, category, image_url, is_available, created_at, updated_at
		FROM menu_items
		WHERE category = $1 AND is_available = TRUE
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query menu items by category: %w", err)
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		var imageURL *string

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Category,
			&imageURL,
			&item.IsAvailable,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan menu item: %w", err)
		}

		if imageURL != nil {
			item.ImageURL = *imageURL
		}

		items = append(items, item)
	}

	return items, nil
}
