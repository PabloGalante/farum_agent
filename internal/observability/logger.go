package observability

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
)

// basic global logger, JSON to stdout.
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func Logger() *slog.Logger {
	return logger
}

// WithFields returns a logger with additional fields.
func WithFields(kv ...any) *slog.Logger {
	return logger.With(kv...)
}

// WithRequestID stores a request_id in the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, requestID)
}

// LoggerFromContext adds request_id if present.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	reqID, _ := ctx.Value(ctxKeyRequestID).(string)
	if reqID == "" {
		return logger
	}
	return logger.With("request_id", reqID)
}
