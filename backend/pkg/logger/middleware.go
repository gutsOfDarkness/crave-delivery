// Package logger provides the Fiber middleware for structured request logging.
// Every request gets a unique Request-ID for distributed tracing.
package logger

import (
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestIDHeader is the header key for request ID propagation
const RequestIDHeader = "X-Request-ID"

// ContextKeyRequestID is the context key for storing request ID
const ContextKeyRequestID = "request_id"

// ContextKeyLogger is the context key for storing request-scoped logger
const ContextKeyLogger = "logger"

// FiberMiddleware returns a Fiber middleware that:
// 1. Generates or propagates Request-ID for every request
// 2. Logs request completion with all required fields
// 3. Captures stack traces for 500 errors
// 4. Attaches request-scoped logger to context
func FiberMiddleware(log *Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		startTime := time.Now()

		// Generate or use existing Request-ID
		// Allows distributed tracing when ID is passed from upstream services
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set Request-ID in response headers for client correlation
		c.Set(RequestIDHeader, requestID)

		// Store request ID in context for downstream handlers
		c.Locals(ContextKeyRequestID, requestID)

		// Create request-scoped logger with Request-ID attached
		requestLogger := log.WithRequestID(requestID)
		c.Locals(ContextKeyLogger, requestLogger)

		// Defer panic recovery to capture stack traces
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				requestLogger.LogPanic(r, stack)

				// Return 500 error
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":      "Internal Server Error",
					"request_id": requestID,
				})

				// Log the failed request
				logRequestCompletion(requestLogger, c, startTime, fiber.StatusInternalServerError, "panic recovered")
			}
		}()

		// Process the request
		err := c.Next()

		// Determine status code and error message
		statusCode := c.Response().StatusCode()
		var errorMsg string

		if err != nil {
			// Fiber error handling
			if e, ok := err.(*fiber.Error); ok {
				statusCode = e.Code
				errorMsg = e.Message
			} else {
				statusCode = fiber.StatusInternalServerError
				errorMsg = err.Error()
			}
		}

		// Log request completion
		logRequestCompletion(requestLogger, c, startTime, statusCode, errorMsg)

		return err
	}
}

// logRequestCompletion logs the complete request/response cycle
func logRequestCompletion(log *Logger, c *fiber.Ctx, startTime time.Time, statusCode int, errorMsg string) {
	entry := RequestLogEntry{
		Timestamp:  time.Now(),
		RequestID:  c.Locals(ContextKeyRequestID).(string),
		Method:     c.Method(),
		Path:       c.Path(),
		StatusCode: statusCode,
		Latency:    time.Since(startTime),
		ClientIP:   c.IP(),
		UserAgent:  c.Get("User-Agent"),
		Error:      errorMsg,
	}

	// For 500 errors, include additional context
	if statusCode >= 500 {
		log.LogRequest(entry)
		// Stack trace is automatically added by zap for error level logs
	} else {
		log.LogRequest(entry)
	}
}

// GetRequestLogger retrieves the request-scoped logger from Fiber context.
// Use this in handlers to get a logger with Request-ID already attached.
func GetRequestLogger(c *fiber.Ctx) *Logger {
	if logger, ok := c.Locals(ContextKeyLogger).(*Logger); ok {
		return logger
	}
	// Fallback to new logger if not found (shouldn't happen with middleware)
	return NewLogger()
}

// GetRequestID retrieves the Request-ID from Fiber context.
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(ContextKeyRequestID).(string); ok {
		return id
	}
	return ""
}
