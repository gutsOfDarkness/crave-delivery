-- Migration: 001_initial_schema
-- Description: Creates core tables for food delivery system
-- Author: System Architect
-- Date: 2024-01-15

-- Enable UUID extension for generating UUIDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- ENUM TYPES
-- ============================================================================

-- Order status enum with all possible states in the order lifecycle
-- State machine: PENDING -> AWAITING_PAYMENT -> PAID/PAYMENT_FAILED -> ACCEPTED -> DELIVERED
CREATE TYPE order_status AS ENUM (
    'PENDING',           -- Order created, items selected
    'AWAITING_PAYMENT',  -- Razorpay order created, waiting for payment
    'PAYMENT_FAILED',    -- Payment attempt failed
    'PAID',              -- Payment verified and confirmed
    'ACCEPTED',          -- Restaurant accepted the order
    'DELIVERED'          -- Order delivered to customer
);

-- ============================================================================
-- USERS TABLE
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Phone number is the primary identifier for Indian users
    -- Indexed and unique for fast OTP-based login lookups
    phone_number VARCHAR(15) NOT NULL,
    
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    
    -- Role flag for admin access
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timestamps for auditing
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT users_phone_number_unique UNIQUE (phone_number),
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_phone_number_format CHECK (phone_number ~ '^\+?[0-9]{10,14}$')
);

-- Index on phone_number for fast login lookups
-- B-tree index is optimal for equality comparisons
CREATE INDEX idx_users_phone_number ON users(phone_number);

-- Index on created_at for time-based queries (e.g., new user analytics)
CREATE INDEX idx_users_created_at ON users(created_at);

-- ============================================================================
-- MENU_ITEMS TABLE
-- ============================================================================

CREATE TABLE menu_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    name VARCHAR(200) NOT NULL,
    description TEXT,
    
    -- Price stored in PAISA (1/100 of Rupee) as INTEGER
    -- This avoids floating-point precision errors in financial calculations
    -- Example: ₹150.50 is stored as 15050
    price INTEGER NOT NULL,
    
    -- Category for menu organization (e.g., 'starters', 'main_course', 'beverages')
    category VARCHAR(50) NOT NULL,
    
    -- URL to item image (stored in CDN/S3)
    image_url TEXT,
    
    -- Availability toggle for out-of-stock items
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    -- Price must be positive (at least 1 paisa)
    CONSTRAINT menu_items_price_positive CHECK (price > 0),
    CONSTRAINT menu_items_name_not_empty CHECK (LENGTH(TRIM(name)) > 0)
);

-- Index on category for filtered menu queries
CREATE INDEX idx_menu_items_category ON menu_items(category);

-- Index on is_available for filtering available items
CREATE INDEX idx_menu_items_available ON menu_items(is_available) WHERE is_available = TRUE;

-- Composite index for common query pattern: available items by category
CREATE INDEX idx_menu_items_category_available ON menu_items(category, is_available);

-- ============================================================================
-- ORDERS TABLE
-- ============================================================================

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Foreign key to users table
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    
    -- Order status using enum type
    status order_status NOT NULL DEFAULT 'PENDING',
    
    -- Total amount in PAISA (calculated server-side, never trust client)
    -- Must always be positive
    total_amount INTEGER NOT NULL,
    
    -- Razorpay integration fields
    -- razorpay_order_id: Created when initiating payment
    -- razorpay_payment_id: Set after successful payment verification
    razorpay_order_id VARCHAR(50),
    razorpay_payment_id VARCHAR(50),
    
    -- Version field for OPTIMISTIC LOCKING
    -- Prevents race conditions in concurrent updates
    -- Every UPDATE must check: WHERE version = current_version
    -- And increment: SET version = version + 1
    version INTEGER NOT NULL DEFAULT 1,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT orders_total_amount_positive CHECK (total_amount > 0),
    CONSTRAINT orders_razorpay_order_id_unique UNIQUE (razorpay_order_id),
    CONSTRAINT orders_razorpay_payment_id_unique UNIQUE (razorpay_payment_id)
);

-- Index on user_id for fetching user's order history
CREATE INDEX idx_orders_user_id ON orders(user_id);

-- Index on status for filtering orders by state
CREATE INDEX idx_orders_status ON orders(status);

-- Index on razorpay_order_id for webhook lookups (unique, so B-tree is optimal)
CREATE INDEX idx_orders_razorpay_order_id ON orders(razorpay_order_id) WHERE razorpay_order_id IS NOT NULL;

-- Composite index for user's orders by status (common query pattern)
CREATE INDEX idx_orders_user_status ON orders(user_id, status);

-- Index on created_at for time-based queries and analytics
CREATE INDEX idx_orders_created_at ON orders(created_at);

-- ============================================================================
-- ORDER_ITEMS TABLE (Line items for each order)
-- ============================================================================

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Foreign key to orders table
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Foreign key to menu_items (RESTRICT delete to preserve order history)
    menu_item_id UUID NOT NULL REFERENCES menu_items(id) ON DELETE RESTRICT,
    
    -- Snapshot of item name at time of order (menu item name might change)
    name VARCHAR(200) NOT NULL,
    
    -- Price at time of order in PAISA (snapshot, menu price might change)
    price INTEGER NOT NULL,
    
    -- Quantity ordered
    quantity INTEGER NOT NULL,
    
    -- Timestamp
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT order_items_price_positive CHECK (price > 0),
    CONSTRAINT order_items_quantity_positive CHECK (quantity > 0)
);

-- Index on order_id for fetching order items
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- ============================================================================
-- WEBHOOK_LOGS TABLE (Audit trail for payment webhooks)
-- ============================================================================

CREATE TABLE webhook_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source of webhook (e.g., 'razorpay')
    source VARCHAR(50) NOT NULL,
    
    -- Event type (e.g., 'payment.captured', 'payment.failed')
    event_type VARCHAR(100) NOT NULL,
    
    -- Full payload stored as JSONB for flexibility
    payload JSONB NOT NULL,
    
    -- Signature verification result
    signature_valid BOOLEAN NOT NULL,
    
    -- Processing result
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    processing_error TEXT,
    
    -- Related order (if identified)
    order_id UUID REFERENCES orders(id),
    
    -- Timestamp
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index on source and event_type for filtering logs
CREATE INDEX idx_webhook_logs_source_event ON webhook_logs(source, event_type);

-- Index on order_id for finding webhooks related to an order
CREATE INDEX idx_webhook_logs_order_id ON webhook_logs(order_id) WHERE order_id IS NOT NULL;

-- Index on created_at for time-based queries
CREATE INDEX idx_webhook_logs_created_at ON webhook_logs(created_at);

-- ============================================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for users table
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for menu_items table
CREATE TRIGGER trigger_menu_items_updated_at
    BEFORE UPDATE ON menu_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for orders table
CREATE TRIGGER trigger_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS (Documentation)
-- ============================================================================

COMMENT ON TABLE users IS 'Registered users with phone-based authentication';
COMMENT ON TABLE menu_items IS 'Food items available for ordering. Price in paisa.';
COMMENT ON TABLE orders IS 'Customer orders with Razorpay payment integration';
COMMENT ON TABLE order_items IS 'Line items for each order with price snapshot';
COMMENT ON TABLE webhook_logs IS 'Audit trail for all payment webhook attempts';

COMMENT ON COLUMN orders.version IS 'Optimistic locking version. Increment on every update.';
COMMENT ON COLUMN orders.total_amount IS 'Total in paisa (1/100 rupee). Always calculated server-side.';
COMMENT ON COLUMN menu_items.price IS 'Price in paisa (1/100 rupee) to avoid floating-point errors.';
