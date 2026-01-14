// Package database provides PostgreSQL connection management using pgx.
// Implements singleton connection pool with auto-reconnect capabilities.
package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fooddelivery/pkg/logger"
)

// Pool wraps pgxpool.Pool with additional functionality
// like health checks and reconnection logic.
type Pool struct {
	*pgxpool.Pool
	log      *logger.Logger
	connStr  string
	mu       sync.RWMutex
	isHealthy bool
}

// Singleton instance for the database pool
var (
	instance *Pool
	once     sync.Once
)

// NewPostgresPool creates a singleton PostgreSQL connection pool.
// Uses sync.Once to ensure only one pool exists across the application.
// This prevents connection exhaustion and ensures consistent pool management.
func NewPostgresPool(ctx context.Context, connStr string, log *logger.Logger) (*Pool, error) {
	var initErr error

	once.Do(func() {
		pool, err := createPool(ctx, connStr, log)
		if err != nil {
			initErr = err
			return
		}
		instance = pool
	})

	if initErr != nil {
		return nil, initErr
	}

	return instance, nil
}

// createPool initializes the actual connection pool with optimized settings
func createPool(ctx context.Context, connStr string, log *logger.Logger) (*Pool, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Pool configuration optimized for 50-500 concurrent users
	// MaxConns = expected_connections * 1.5 for headroom
	config.MaxConns = 50
	config.MinConns = 10

	// Connection lifetime prevents stale connections
	// Connections are recycled after 1 hour to handle DNS changes, etc.
	config.MaxConnLifetime = 1 * time.Hour

	// Idle timeout closes unused connections to free resources
	config.MaxConnIdleTime = 30 * time.Minute

	// Health check interval ensures connections are valid
	config.HealthCheckPeriod = 30 * time.Second

	// Connection timeout prevents hanging on network issues
	config.ConnConfig.ConnectTimeout = 10 * time.Second

	// Before acquire hook for connection validation
	config.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		// Return true if connection is usable, false to discard
		return conn.Ping(ctx) == nil
	}

	// After connect hook for connection setup
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		log.Debug("New database connection established")
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection with ping
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("PostgreSQL connection pool established",
		"max_conns", config.MaxConns,
		"min_conns", config.MinConns,
	)

	p := &Pool{
		Pool:      pool,
		log:       log,
		connStr:   connStr,
		isHealthy: true,
	}

	// Start background health checker with auto-reconnect
	go p.healthChecker(ctx)

	return p, nil
}

// healthChecker runs periodic health checks and attempts reconnection on failure.
// Uses exponential backoff to avoid overwhelming the database during outages.
func (p *Pool) healthChecker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := p.Pool.Ping(ctx)
			if err != nil {
				p.mu.Lock()
				p.isHealthy = false
				p.mu.Unlock()

				p.log.Error("Database health check failed", "error", err)

				// Attempt reconnection with exponential backoff
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(backoff):
						if err := p.Pool.Ping(ctx); err == nil {
							p.mu.Lock()
							p.isHealthy = true
							p.mu.Unlock()
							p.log.Info("Database connection restored")
							backoff = time.Second // Reset backoff
							break
						}

						p.log.Warn("Database reconnection attempt failed",
							"next_retry_in", backoff.String())

						// Exponential backoff with cap
						backoff *= 2
						if backoff > maxBackoff {
							backoff = maxBackoff
						}
					}
				}
			} else {
				p.mu.Lock()
				p.isHealthy = true
				p.mu.Unlock()
			}
		}
	}
}

// IsHealthy returns current health status of the database connection.
// Used by health check endpoints and circuit breakers.
func (p *Pool) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isHealthy
}

// ExecTx executes a function within a database transaction.
// Automatically handles commit/rollback based on error return.
// Uses serializable isolation for critical operations like payments.
func (p *Pool) ExecTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := p.Pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback - no-op if commit succeeds
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				p.log.Error("Failed to rollback transaction", "error", rbErr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecTxWithIsolation executes a function within a transaction with specified isolation level.
// Use ReadCommitted for read-heavy operations, Serializable for payment processing.
func (p *Pool) ExecTxWithIsolation(ctx context.Context, isoLevel pgx.TxIsoLevel, fn func(tx pgx.Tx) error) error {
	tx, err := p.Pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: isoLevel,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				p.log.Error("Failed to rollback transaction", "error", rbErr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Querier interface for abstracting database operations
// Allows both Pool and Tx to be used interchangeably in repositories
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}
