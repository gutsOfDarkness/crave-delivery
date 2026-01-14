// Package usecase implements menu business logic with Redis caching
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"fooddelivery/internal/domain"
	"fooddelivery/internal/repository"
	"fooddelivery/pkg/logger"
	"fooddelivery/pkg/redis"
)

// MenuUsecase handles menu-related business logic
type MenuUsecase struct {
	menuRepo    *repository.MenuRepository
	redisClient *redis.Client
	log         *logger.Logger
}

// NewMenuUsecase creates a new menu usecase
func NewMenuUsecase(menuRepo *repository.MenuRepository, redisClient *redis.Client, log *logger.Logger) *MenuUsecase {
	return &MenuUsecase{
		menuRepo:    menuRepo,
		redisClient: redisClient,
		log:         log,
	}
}

// MenuResponse wraps menu items with metadata
type MenuResponse struct {
	Items      []domain.MenuItem `json:"items"`
	Categories []string          `json:"categories"`
	CacheHit   bool              `json:"cache_hit"`
}

// GetMenu retrieves the full menu with Redis caching.
// Strategy:
// 1. Check Redis cache (key: app:menu:all)
// 2. On HIT: Return cached JSON immediately (fast path)
// 3. On MISS: Query PostgreSQL -> Serialize -> Cache with 1 hour TTL -> Return
func (u *MenuUsecase) GetMenu(ctx context.Context) (*MenuResponse, error) {
	// Step 1: Try Redis cache first
	if u.redisClient != nil {
		var cachedMenu MenuResponse
		found, err := u.redisClient.GetJSON(ctx, redis.MenuCacheKey, &cachedMenu)
		if err != nil {
			// Log but don't fail - cache is optional optimization
			u.log.Warn("Failed to read menu from cache", "error", err)
		} else if found {
			u.log.Debug("Menu cache HIT")
			cachedMenu.CacheHit = true
			return &cachedMenu, nil
		}
	}

	u.log.Debug("Menu cache MISS, querying database")

	// Step 2: Query database
	items, err := u.menuRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch menu: %w", err)
	}

	// Extract unique categories
	categorySet := make(map[string]struct{})
	for _, item := range items {
		categorySet[item.Category] = struct{}{}
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}

	response := &MenuResponse{
		Items:      items,
		Categories: categories,
		CacheHit:   false,
	}

	// Step 3: Cache the response
	if u.redisClient != nil {
		if err := u.redisClient.SetJSON(ctx, redis.MenuCacheKey, response, redis.MenuCacheTTL); err != nil {
			u.log.Warn("Failed to cache menu", "error", err)
			// Don't fail - cache is optimization
		} else {
			u.log.Debug("Menu cached successfully", "ttl", redis.MenuCacheTTL)
		}
	}

	return response, nil
}

// GetMenuItem retrieves a single menu item by ID
func (u *MenuUsecase) GetMenuItem(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error) {
	item, err := u.menuRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// CreateMenuItem creates a new menu item (admin only)
func (u *MenuUsecase) CreateMenuItem(ctx context.Context, item *domain.MenuItem) error {
	if err := u.menuRepo.Create(ctx, item); err != nil {
		return fmt.Errorf("failed to create menu item: %w", err)
	}

	// Invalidate cache after creation
	u.invalidateCache(ctx)

	return nil
}

// UpdateMenuItem updates an existing menu item (admin only)
func (u *MenuUsecase) UpdateMenuItem(ctx context.Context, item *domain.MenuItem) error {
	if err := u.menuRepo.Update(ctx, item); err != nil {
		return err
	}

	// Invalidate cache after update
	u.invalidateCache(ctx)

	return nil
}

// DeleteMenuItem soft-deletes a menu item (admin only)
func (u *MenuUsecase) DeleteMenuItem(ctx context.Context, id uuid.UUID) error {
	if err := u.menuRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache after deletion
	u.invalidateCache(ctx)

	return nil
}

// InvalidateMenuCache explicitly invalidates the menu cache.
// Called by admin endpoint POST /admin/menu/invalidate-cache
func (u *MenuUsecase) InvalidateMenuCache(ctx context.Context) error {
	u.invalidateCache(ctx)
	return nil
}

// invalidateCache removes the menu cache from Redis
func (u *MenuUsecase) invalidateCache(ctx context.Context) {
	if u.redisClient == nil {
		return
	}

	if err := u.redisClient.DeleteKey(ctx, redis.MenuCacheKey); err != nil {
		u.log.Warn("Failed to invalidate menu cache", "error", err)
	} else {
		u.log.Info("Menu cache invalidated")
	}
}

// GetMenuByCategory retrieves menu items filtered by category
func (u *MenuUsecase) GetMenuByCategory(ctx context.Context, category string) ([]domain.MenuItem, error) {
	items, err := u.menuRepo.GetByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch menu by category: %w", err)
	}
	return items, nil
}
