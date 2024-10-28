//go:build go1.21

package fauna

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	LogResponse(ctx context.Context, requestBody []byte, r *http.Response)
}

type ClientLogger struct {
	Logger

	logger *slog.Logger
}

func (d ClientLogger) Debug(msg string, args ...any) {
	if d.logger == nil {
		return
	}

	d.logger.Debug(msg, args...)
}

func (d ClientLogger) Info(msg string, args ...any) {
	if d.logger == nil {
		return
	}

	d.logger.Info(msg, args...)
}

func (d ClientLogger) Warn(msg string, args ...any) {
	if d.logger == nil {
		return
	}

	d.logger.Warn(msg, args...)
}

func (d ClientLogger) Error(msg string, args ...any) {
	if d.logger == nil {
		return
	}

	d.logger.Error(msg, args...)
}

func (d ClientLogger) LogResponse(ctx context.Context, requestBody []byte, r *http.Response) {
	if d.logger == nil {
		return
	}

	requestLogger := d.logger.With(
		slog.String("method", r.Request.Method),
		slog.String("url", r.Request.URL.String()),
		slog.Int("status", r.StatusCode))

	headers := r.Request.Header
	if _, found := headers["Authorization"]; found {
		headers["Authorization"] = []string{"hidden"}
	}
	if d.logger.Enabled(ctx, slog.LevelDebug) {
		requestLogger = requestLogger.With(
			slog.String("requestBody", string(requestBody)),
		)
	}

	requestLogger.With(
		slog.Any("headers", headers)).Info("HTTP Response")
}

// DefaultLogger returns the default logger
func DefaultLogger() Logger {
	clientLogger := ClientLogger{}

	if val, found := os.LookupEnv(EnvFaunaDebug); found {
		if level, _ := strconv.Atoi(val); level >= -4 {
			clientLogger.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.Level(level),
			}))
		}
	}

	return clientLogger
}
