package log

import (
	"context"
	"log"
)

type StandardLogger struct{}

// Error - logs an error message.
func (s StandardLogger) Error(ctx context.Context, msg string, any ...any) {
	log.Printf("[ERROR] "+msg, any...)
}

// Warn - logs a warning message.
func (s StandardLogger) Warn(ctx context.Context, msg string, any ...any) {
	log.Printf("[WARN] "+msg, any...)
}

// Info - logs an informational message.
func (s StandardLogger) Info(ctx context.Context, msg string, any ...any) {
	log.Printf("[INFO] "+msg, any...)
}

// Debug - logs a debug message.
func (s StandardLogger) Debug(ctx context.Context, msg string, any ...any) {
	log.Printf("[DEBUG] "+msg, any...)
}
