// Package logger provides structured JSON logging using Uber's Zap library.
// All logs include timestamp, level, and contextual fields for observability.
// Designed for high-performance with minimal allocations.
package logger

import (
	"os"
	"runtime"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger to provide a simplified interface
// while maintaining full structured logging capabilities.
type Logger struct {
	*zap.SugaredLogger
	baseLogger *zap.Logger
}

// NewLogger creates a production-ready JSON logger.
// Output format: {"level":"info","ts":"2024-01-15T10:30:00Z","msg":"...","key":"value"}
func NewLogger() *Logger {
	// Configure encoder for JSON output
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Determine log level from environment
	logLevel := getLogLevel()

	// Create core with JSON encoder writing to stdout
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		logLevel,
	)

	// Build logger with caller info and stack traces for errors
	baseLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1), // Skip wrapper function in call stack
		zap.AddStacktrace(zapcore.ErrorLevel), // Auto-add stack trace for errors
	)

	return &Logger{
		SugaredLogger: baseLogger.Sugar(),
		baseLogger:    baseLogger,
	}
}

// getLogLevel returns the appropriate zap level based on LOG_LEVEL env var
func getLogLevel() zapcore.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// WithRequestID creates a new logger with the request ID field attached.
// Used to correlate all logs within a single request lifecycle.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.With("request_id", requestID),
		baseLogger:    l.baseLogger.With(zap.String("request_id", requestID)),
	}
}

// WithFields creates a new logger with additional contextual fields.
// Useful for adding user_id, order_id, etc. to a request-scoped logger.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		zapFields = append(zapFields, k, v)
	}
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(zapFields...),
		baseLogger:    l.baseLogger,
	}
}

// LogError logs an error with full stack trace.
// Use this for unexpected errors that need investigation.
func (l *Logger) LogError(msg string, err error, fields ...interface{}) {
	// Capture stack trace manually for detailed debugging
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	stackTrace := string(buf[:n])

	allFields := append(fields, "error", err.Error(), "stack", stackTrace)
	l.SugaredLogger.Errorw(msg, allFields...)
}

// LogPanic logs a panic with recovery info.
// Called from recovery middleware to capture panic details.
func (l *Logger) LogPanic(recovered interface{}, stack []byte) {
	l.SugaredLogger.Errorw("Panic recovered",
		"panic", recovered,
		"stack", string(stack),
	)
}

// Sync flushes any buffered log entries.
// Should be called before application exit.
func (l *Logger) Sync() {
	_ = l.baseLogger.Sync()
}

// RequestLogEntry represents the structured data for a request log
type RequestLogEntry struct {
	Timestamp  time.Time     `json:"timestamp"`
	Level      string        `json:"level"`
	RequestID  string        `json:"request_id"`
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency_ms"`
	ClientIP   string        `json:"client_ip"`
	UserAgent  string        `json:"user_agent"`
	Error      string        `json:"error,omitempty"`
}

// LogRequest logs a complete request/response cycle with all required fields.
// Called by the logging middleware after request completion.
func (l *Logger) LogRequest(entry RequestLogEntry) {
	level := "info"
	if entry.StatusCode >= 500 {
		level = "error"
	} else if entry.StatusCode >= 400 {
		level = "warn"
	}

	fields := []interface{}{
		"request_id", entry.RequestID,
		"method", entry.Method,
		"path", entry.Path,
		"status_code", entry.StatusCode,
		"latency_ms", entry.Latency.Milliseconds(),
		"client_ip", entry.ClientIP,
		"user_agent", entry.UserAgent,
	}

	if entry.Error != "" {
		fields = append(fields, "error", entry.Error)
	}

	switch level {
	case "error":
		l.SugaredLogger.Errorw("Request completed", fields...)
	case "warn":
		l.SugaredLogger.Warnw("Request completed", fields...)
	default:
		l.SugaredLogger.Infow("Request completed", fields...)
	}
}
