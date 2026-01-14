// Package repository implements order data access with optimistic locking
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"fooddelivery/internal/domain"
	"fooddelivery/pkg/database"
)

// OrderRepository handles order data persistence
type OrderRepository struct {
	db *database.Pool
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *database.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts a new order with its items in a transaction
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	return r.db.ExecTx(ctx, func(tx pgx.Tx) error {
		// Insert order
		orderQuery := `
			INSERT INTO orders (id, user_id, status, total_amount, razorpay_order_id, version, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		order.ID = uuid.New()
		order.Version = 1
		now := time.Now()
		order.CreatedAt = now
		order.UpdatedAt = now

		_, err := tx.Exec(ctx, orderQuery,
			order.ID,
			order.UserID,
			order.Status,
			order.TotalAmount,
			order.RazorpayOrderID,
			order.Version,
			order.CreatedAt,
			order.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order: %w", err)
		}

		// Insert order items
		itemQuery := `
			INSERT INTO order_items (id, order_id, menu_item_id, name, price, quantity, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

		for i := range order.Items {
			order.Items[i].ID = uuid.New()
			order.Items[i].OrderID = order.ID
			order.Items[i].CreatedAt = now

			_, err := tx.Exec(ctx, itemQuery,
				order.Items[i].ID,
				order.Items[i].OrderID,
				order.Items[i].MenuItemID,
				order.Items[i].Name,
				order.Items[i].Price,
				order.Items[i].Quantity,
				order.Items[i].CreatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to insert order item: %w", err)
			}
		}

		return nil
	})
}

// GetByID retrieves an order with its items
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	orderQuery := `
		SELECT id, user_id, status, total_amount, razorpay_order_id, razorpay_payment_id, version, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	order := &domain.Order{}
	var razorpayOrderID, razorpayPaymentID *string

	err := r.db.QueryRow(ctx, orderQuery, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&razorpayOrderID,
		&razorpayPaymentID,
		&order.Version,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if razorpayOrderID != nil {
		order.RazorpayOrderID = *razorpayOrderID
	}
	if razorpayPaymentID != nil {
		order.RazorpayPaymentID = *razorpayPaymentID
	}

	// Fetch order items
	items, err := r.getOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

// GetByRazorpayOrderID retrieves an order by Razorpay order ID
// Used by webhook handler to find the order for payment updates
func (r *OrderRepository) GetByRazorpayOrderID(ctx context.Context, razorpayOrderID string) (*domain.Order, error) {
	orderQuery := `
		SELECT id, user_id, status, total_amount, razorpay_order_id, razorpay_payment_id, version, created_at, updated_at
		FROM orders
		WHERE razorpay_order_id = $1
	`

	order := &domain.Order{}
	var rpOrderID, rpPaymentID *string

	err := r.db.QueryRow(ctx, orderQuery, razorpayOrderID).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&rpOrderID,
		&rpPaymentID,
		&order.Version,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get order by razorpay ID: %w", err)
	}

	if rpOrderID != nil {
		order.RazorpayOrderID = *rpOrderID
	}
	if rpPaymentID != nil {
		order.RazorpayPaymentID = *rpPaymentID
	}

	return order, nil
}

// GetByUserID retrieves all orders for a user
func (r *OrderRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	query := `
		SELECT id, user_id, status, total_amount, razorpay_order_id, razorpay_payment_id, version, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		var razorpayOrderID, razorpayPaymentID *string

		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.TotalAmount,
			&razorpayOrderID,
			&razorpayPaymentID,
			&order.Version,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		if razorpayOrderID != nil {
			order.RazorpayOrderID = *razorpayOrderID
		}
		if razorpayPaymentID != nil {
			order.RazorpayPaymentID = *razorpayPaymentID
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// UpdateStatus updates order status with optimistic locking
// This is critical for payment processing to prevent race conditions
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus domain.OrderStatus, expectedVersion int) error {
	// OPTIMISTIC LOCKING: Only update if version matches expected version
	// This prevents race conditions where two concurrent requests try to update the same order
	// If version doesn't match, another request already modified the order
	query := `
		UPDATE orders
		SET status = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $3
	`

	result, err := r.db.Exec(ctx, query, orderID, newStatus, expectedVersion)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// If no rows affected, either order doesn't exist or version mismatch
	if result.RowsAffected() == 0 {
		// Check if order exists
		_, err := r.GetByID(ctx, orderID)
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		// Order exists but version mismatch - concurrent modification
		return ErrVersionConflict
	}

	return nil
}

// UpdatePaymentStatus updates order with payment information atomically
// Uses SERIALIZABLE isolation to ensure payment is recorded exactly once
func (r *OrderRepository) UpdatePaymentStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus, paymentID string, expectedVersion int) error {
	return r.db.ExecTxWithIsolation(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		// First, check current status to prevent double processing
		var currentStatus domain.OrderStatus
		var currentVersion int

		checkQuery := `
			SELECT status, version FROM orders WHERE id = $1 FOR UPDATE
		`
		err := tx.QueryRow(ctx, checkQuery, orderID).Scan(&currentStatus, &currentVersion)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("failed to check order status: %w", err)
		}

		// Verify version matches (optimistic lock check)
		if currentVersion != expectedVersion {
			return ErrVersionConflict
		}

		// Prevent processing if already in a terminal state
		if currentStatus == domain.OrderStatusPaid || currentStatus == domain.OrderStatusDelivered {
			// Already processed, idempotent success
			return nil
		}

		// Update order with payment ID
		updateQuery := `
			UPDATE orders
			SET status = $2, razorpay_payment_id = $3, version = version + 1, updated_at = NOW()
			WHERE id = $1
		`

		_, err = tx.Exec(ctx, updateQuery, orderID, status, paymentID)
		if err != nil {
			return fmt.Errorf("failed to update payment status: %w", err)
		}

		return nil
	})
}

// SetRazorpayOrderID updates the Razorpay order ID for an order
func (r *OrderRepository) SetRazorpayOrderID(ctx context.Context, orderID uuid.UUID, razorpayOrderID string, expectedVersion int) error {
	query := `
		UPDATE orders
		SET razorpay_order_id = $2, status = $3, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $4
	`

	result, err := r.db.Exec(ctx, query, orderID, razorpayOrderID, domain.OrderStatusAwaitingPayment, expectedVersion)
	if err != nil {
		return fmt.Errorf("failed to set razorpay order ID: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrVersionConflict
	}

	return nil
}

// getOrderItems retrieves all items for an order
func (r *OrderRepository) getOrderItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	query := `
		SELECT id, order_id, menu_item_id, name, price, quantity, created_at
		FROM order_items
		WHERE order_id = $1
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.MenuItemID,
			&item.Name,
			&item.Price,
			&item.Quantity,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// GetAllOrders retrieves all orders (admin only)
func (r *OrderRepository) GetAllOrders(ctx context.Context, limit, offset int) ([]domain.Order, error) {
	query := `
		SELECT id, user_id, status, total_amount, razorpay_order_id, razorpay_payment_id, version, created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query all orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		var razorpayOrderID, razorpayPaymentID *string

		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.TotalAmount,
			&razorpayOrderID,
			&razorpayPaymentID,
			&order.Version,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		if razorpayOrderID != nil {
			order.RazorpayOrderID = *razorpayOrderID
		}
		if razorpayPaymentID != nil {
			order.RazorpayPaymentID = *razorpayPaymentID
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// LogWebhook stores webhook attempt for audit trail
func (r *OrderRepository) LogWebhook(ctx context.Context, source, eventType string, payload []byte, signatureValid bool, orderID *uuid.UUID, processingError string) error {
	query := `
		INSERT INTO webhook_logs (id, source, event_type, payload, signature_valid, processed, processing_error, order_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	processed := processingError == ""

	_, err := r.db.Exec(ctx, query,
		uuid.New(),
		source,
		eventType,
		payload,
		signatureValid,
		processed,
		processingError,
		orderID,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to log webhook: %w", err)
	}

	return nil
}
