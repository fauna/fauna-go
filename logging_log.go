//go:build !go1.21

package fauna

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

type DriverLogger interface {
	Info(msg string)
	LogResponse(ctx context.Context, requestBody []byte, r *http.Response)
}

type ClientLogger struct {
	DriverLogger

	logger *log.Logger
	level  int
}

func (d ClientLogger) Debug(msg string) {
	if d.logger == nil {
		return
	}

	d.logger.Print("DEBUG: " + msg)
}

func (d ClientLogger) Info(msg string) {
	if d.logger == nil {
		return
	}

	d.logger.Print("INFO: " + msg)
}

func (d ClientLogger) Warn(msg string) {
	if d.logger == nil {
		return
	}

	d.logger.Print("WARN: " + msg)
}

func (d ClientLogger) Error(msg string) {
	if d.logger == nil {
		return
	}

	d.logger.Print("ERROR: " + msg)
}

func (d ClientLogger) LogResponse(ctx context.Context, requestBody []byte, r *http.Response) {
	if d.logger == nil {
		return
	}

	headers := r.Request.Header
	if d.level > -4 {
		if _, found := headers["Authorization"]; found {
			headers["Authorization"] = []string{"hidden"}
		}
	}

	d.Debug(fmt.Sprintf("Request Body: %s", string(requestBody)))
	d.Info(fmt.Sprintf("HTTP Response - Status: %s, From: %s, Headers: %v", r.Status, r.Request.URL.String(), headers))
}

// DefaultLogger returns the default logger
func DefaultLogger() DriverLogger {
	clientLogger := ClientLogger{}

	if val, found := os.LookupEnv(EnvFaunaDebug); found {
		if level, _ := strconv.Atoi(val); level >= -4 {
			clientLogger.level = level
			clientLogger.logger = log.New(os.Stdout, "[fauna-go] ", log.LstdFlags|log.Lshortfile)
		}
	}

	return clientLogger
}
