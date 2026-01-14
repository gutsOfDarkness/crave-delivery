// Package usecase implements order business logic
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"fooddelivery/internal/domain"
	"fooddelivery/internal/repository"
	"fooddelivery/pkg/logger"
)

// OrderUsecase handles order-related business logic
type OrderUsecase struct {
	orderRepo      *repository.OrderRepository
	paymentUsecase *PaymentUsecase
	log            *logger.Logger
}

// NewOrderUsecase creates a new order usecase
func NewOrderUsecase(orderRepo *repository.OrderRepository, paymentUsecase *PaymentUsecase, log *logger.Logger) *OrderUsecase {
	return &OrderUsecase{
		orderRepo:      orderRepo,
		paymentUsecase: paymentUsecase,
		log:            log,
	}
}

// GetOrder retrieves an order by ID
func (u *OrderUsecase) GetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// GetUserOrders retrieves all orders for a user
func (u *OrderUsecase) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	orders, err := u.orderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user orders: %w", err)
	}
	return orders, nil
}

// GetAllOrders retrieves all orders (admin only)
func (u *OrderUsecase) GetAllOrders(ctx context.Context, limit, offset int) ([]domain.Order, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	orders, err := u.orderRepo.GetAllOrders(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all orders: %w", err)
	}
	return orders, nil
}

// UpdateOrderStatus updates order status (admin only)
// Valid transitions: PAID -> ACCEPTED -> DELIVERED
func (u *OrderUsecase) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus domain.OrderStatus) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Validate state transition
	if !isValidStatusTransition(order.Status, newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", order.Status, newStatus)
	}

	if err := u.orderRepo.UpdateStatus(ctx, orderID, newStatus, order.Version); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	u.log.Info("Order status updated",
		"order_id", orderID.String(),
		"old_status", order.Status,
		"new_status", newStatus,
	)

	return nil
}

// isValidStatusTransition checks if status transition is allowed
func isValidStatusTransition(current, next domain.OrderStatus) bool {
	validTransitions := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPending:         {domain.OrderStatusAwaitingPayment, domain.OrderStatusPaymentFailed},
		domain.OrderStatusAwaitingPayment: {domain.OrderStatusPaid, domain.OrderStatusPaymentFailed},
		domain.OrderStatusPaymentFailed:   {domain.OrderStatusAwaitingPayment}, // Allow retry
		domain.OrderStatusPaid:            {domain.OrderStatusAccepted},
		domain.OrderStatusAccepted:        {domain.OrderStatusDelivered},
	}

	allowedNext, ok := validTransitions[current]
	if !ok {
		return false
	}

	for _, allowed := range allowedNext {
		if next == allowed {
			return true
		}
	}
	return false
}
