// Package logging contains the logging utility for Nemesis
package logging

import (
	"context"
	"go.uber.org/zap"
)

var logger zap.Logger

const loggerKey = "context-logger"

func init() {
	var err error

	l, err := zap.NewProduction()
	if err != nil {
		panic("unable to initialize logger: " + err.Error())
	}

	logger = *l
}

// NewContext returns a context with the zap logger with extra fields added
func NewContext(ctx context.Context, fields ...zap.Field) context.Context {
	tempLogger := WithContext(ctx)
	x := tempLogger.With(fields...)
	return context.WithValue(ctx, loggerKey, *x)
}

func WithContext(ctx context.Context) zap.Logger {
	if ctx == nil {
		return logger
	}

	if ctxLogger, ok := ctx.Value(loggerKey).(zap.Logger); ok {
		return ctxLogger
	}

	return logger
}
