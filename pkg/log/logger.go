package log

import (
	"context"
	"log"
)

type Standard struct{}

// Error - logs an error message.
func (s Standard) Error(ctx context.Context, msg string, any ...any) {
	log.Printf("[ERROR] "+msg, any...)
}

// Warn - logs a warning message.
func (s Standard) Warn(ctx context.Context, msg string, any ...any) {
	log.Printf("[WARN] "+msg, any...)
}

// Info - logs an informational message.
func (s Standard) Info(ctx context.Context, msg string, any ...any) {
	log.Printf("[INFO] "+msg, any...)
}

// Debug - logs a debug message.
func (s Standard) Debug(ctx context.Context, msg string, any ...any) {
	log.Printf("[DEBUG] "+msg, any...)
}

type Noop struct{}

func (s Noop) Error(ctx context.Context, msg string, any ...any) {
}

func (s Noop) Warn(ctx context.Context, msg string, any ...any) {
}

func (s Noop) Info(ctx context.Context, msg string, any ...any) {
}

func (s Noop) Debug(ctx context.Context, msg string, any ...any) {
}
