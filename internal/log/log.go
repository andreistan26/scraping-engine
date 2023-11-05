package log

import (
	"context"

	"go.uber.org/zap"
)

/* in order to avoid conflicts in ctx.Value(...) */
type contextKey struct{}

/* add logger to context */
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

/* get logger from context or default logger if not set */
func FromContext(ctx context.Context) *zap.SugaredLogger {
	logger, ok := ctx.Value(contextKey{}).(*zap.SugaredLogger)
	if !ok || logger == nil {
		return zap.S()
	}

	return logger
}
