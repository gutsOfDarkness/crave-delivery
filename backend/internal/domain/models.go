// Package domain contains core business entities and value objects.
// These models are database-agnostic and represent the business domain.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents the state machine for order lifecycle.
// State transitions: PENDING -> AWAITING_PAYMENT -> PAID/PAYMENT_FAILED -> ACCEPTED -> DELIVERED
type OrderStatus string

const (
	OrderStatusPending        OrderStatus = "PENDING"
	OrderStatusAwaitingPayment OrderStatus = "AWAITING_PAYMENT"
	OrderStatusPaymentFailed  OrderStatus = "PAYMENT_FAILED"
	OrderStatusPaid           OrderStatus = "PAID"
	OrderStatusAccepted       OrderStatus = "ACCEPTED"
	OrderStatusDelivered      OrderStatus = "DELIVERED"
)

// User represents a registered user in the system
type User struct {
	ID          uuid.UUID  `json:"id"`
	PhoneNumber string     `json:"phone_number"`
	Name        string     `json:"name"`
	Email       string     `json:"email,omitempty"`
	IsAdmin     bool       `json:"is_admin"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// MenuItem represents a food item available for ordering.
// Price is stored in paisa (1/100 of rupee) to avoid floating point errors.
type MenuItem struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int64     `json:"price"` // Price in paisa (e.g., 10000 = ₹100.00)
	Category    string    `json:"category"`
	ImageURL    string    `json:"image_url,omitempty"`
	IsAvailable bool      `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PriceInRupees returns the price formatted in rupees for display
func (m *MenuItem) PriceInRupees() float64 {
	return float64(m.Price) / 100.0
}

// Order represents a customer order with payment tracking.
// Version field enables optimistic locking to prevent race conditions.
type Order struct {
	ID                uuid.UUID   `json:"id"`
	UserID            uuid.UUID   `json:"user_id"`
	Status            OrderStatus `json:"status"`
	TotalAmount       int64       `json:"total_amount"` // Amount in paisa
	RazorpayOrderID   string      `json:"razorpay_order_id,omitempty"`
	RazorpayPaymentID string      `json:"razorpay_payment_id,omitempty"`
	Version           int         `json:"version"` // For optimistic locking
	Items             []OrderItem `json:"items"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// TotalInRupees returns the total amount formatted in rupees
func (o *Order) TotalInRupees() float64 {
	return float64(o.TotalAmount) / 100.0
}

// OrderItem represents a line item in an order
type OrderItem struct {
	ID         uuid.UUID `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	MenuItemID uuid.UUID `json:"menu_item_id"`
	Name       string    `json:"name"`
	Price      int64     `json:"price"`    // Price at time of order (in paisa)
	Quantity   int       `json:"quantity"`
	CreatedAt  time.Time `json:"created_at"`
}

// Subtotal returns the line item subtotal in paisa
func (oi *OrderItem) Subtotal() int64 {
	return oi.Price * int64(oi.Quantity)
}

// CartItem represents an item in the user's cart (before order creation)
type CartItem struct {
	MenuItemID uuid.UUID `json:"menu_item_id"`
	Quantity   int       `json:"quantity"`
}

// Cart represents the user's shopping cart
type Cart struct {
	UserID uuid.UUID  `json:"user_id"`
	Items  []CartItem `json:"items"`
}

// Session represents a user session stored in Redis
type Session struct {
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
	IsAdmin   bool      `json:"is_admin"`
	ExpiresAt time.Time `json:"expires_at"`
}
