package logging

import (
	"context"
	"github.com/TakeoffTech/go-log/zapx"
	"github.com/TakeoffTech/site-info-svc/common"
	"go.uber.org/zap"
	"log"
	"net/http"
)

var logger *zap.SugaredLogger

// init function will initialise a base logger
func init() {
	zapLogger, err := zapx.New(zapx.Config{
		ServiceName: common.ServiceName,
	})

	if err != nil {
		log.Printf(`{"severity": "error", "message": "failed to initialize zap logging: %v"}`, err)

		logger = zap.S()
	}
	logger = zapLogger
}

type CtxLogger struct{}

// GetLoggerFromContext is used to get a logger from context
// If context based logger is not found base logger is returned
func GetLoggerFromContext(ctx context.Context) *zap.SugaredLogger {
	newLogger := logger
	if l, ok := ctx.Value(CtxLogger{}).(*zap.SugaredLogger); ok {
		return l
	}

	return newLogger
}

// GetContextWithLogger This function accepts a http.Request object, extracts the X-Correlation-ID from request object
// Returns a new context based logger key and newLogger with the X-Correlation-ID
func GetContextWithLogger(request *http.Request) (CtxLogger, *zap.SugaredLogger) {
	newLogger := logger
	if request.Header.Get(common.HeaderXCorrelationID) != "" {
		newLogger = logger.With(common.HeaderXCorrelationID, request.Header.Get(common.HeaderXCorrelationID))
	}

	return CtxLogger{}, newLogger
}

// GetLoggerWithXCorrelationID This function accepts a xCorrelationID string,
// Returns a newLogger with the X-Correlation-ID
func GetLoggerWithXCorrelationID(xCorrelationID string) *zap.SugaredLogger {
	newLogger := logger
	if xCorrelationID != "" {
		newLogger = logger.With(common.HeaderXCorrelationID, xCorrelationID)
	}

	return newLogger
}
