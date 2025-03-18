package log

import "context"

// Logger is an interface to be used as a standard log for.
type Logger interface {
	// Error - is used when something goes wrong and blocks the operation from going on.
	Error(ctx context.Context, msg string, any ...any)

	// Warn - is used when something is different than expected.
	Warn(ctx context.Context, msg string, any ...any)

	// Info - is used for expected and helpful information.
	Info(ctx context.Context, msg string, any ...any)

	// Debug - is used for development and production troubleshooting.
	Debug(ctx context.Context, msg string, any ...any)
}

func Log() Logger {
	return logger
}
