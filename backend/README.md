# Food Delivery Backend

High-performance, low-latency food delivery API built with Go (Fiber framework).

## Architecture

**Modular Monolith** following Clean Architecture:
- `cmd/api` - Application entry point
- `internal/handlers` - HTTP handlers (Delivery layer)
- `internal/usecase` - Business logic (Application layer)
- `internal/repository` - Data access (Infrastructure layer)
- `internal/domain` - Domain models
- `pkg/` - Shared packages (logger, database, redis)

## Tech Stack

- **Framework:** Fiber v2 (high-performance HTTP)
- **Database:** PostgreSQL with pgx driver
- **Cache:** Redis for menu caching & idempotency
- **Logging:** Uber Zap (structured JSON)
- **Payments:** Razorpay integration

## Setup

### Prerequisites
- Go 1.22+
- PostgreSQL 14+
- Redis 7+

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Required variables:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `RAZORPAY_KEY_ID` - Razorpay API key
- `RAZORPAY_KEY_SECRET` - Razorpay secret
- `RAZORPAY_WEBHOOK_SECRET` - Webhook signature secret
- `JWT_SECRET` - JWT signing key

### Database Migration

```bash
psql $DATABASE_URL -f migrations/001_initial_schema.sql
```

### Run Server

```bash
go mod download
go run cmd/api/main.go
```

## API Endpoints

### Public
- `GET /health` - Health check
- `POST /api/v1/auth/register` - Register user
- `POST /api/v1/auth/login` - Request OTP
- `POST /api/v1/auth/verify-otp` - Verify OTP, get JWT
- `GET /api/v1/menu` - Get menu (cached)

### Protected (requires JWT)
- `POST /api/v1/orders/create` - Create order
- `GET /api/v1/orders` - User's orders
- `POST /api/v1/orders/verify` - Verify payment

### Admin
- `POST /api/v1/admin/menu` - Create menu item
- `PUT /api/v1/admin/menu/:id` - Update menu item
- `POST /api/v1/admin/menu/invalidate-cache` - Clear menu cache

### Webhooks
- `POST /webhooks/razorpay` - Razorpay payment webhooks

## Key Features

### Idempotent Order Creation
Orders with identical cart contents within 1 minute return the same Razorpay order ID, preventing duplicate charges.

### Optimistic Locking
Order updates use version-based optimistic locking to prevent race conditions in payment processing.

### Structured Logging
Every request includes:
- Unique Request-ID for tracing
- Timestamp, method, path, status, latency
- Stack traces for 500 errors

### Redis Caching Strategy
Menu items cached for 1 hour with automatic invalidation on updates.

## Security

- All prices calculated server-side (never trust client)
- Razorpay webhook signature verification (HMAC SHA256)
- JWT authentication with expiration
- SQL injection prevention via parameterized queries
