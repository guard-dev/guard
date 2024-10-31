package logger

import (
	"context"

	"github.com/hyperdxio/opentelemetry-go/otelzap"
	sdk "github.com/hyperdxio/opentelemetry-logs-go/sdk/logs"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type LoggerConnectProps struct {
	Production     bool
	LoggerProvider *sdk.LoggerProvider
}

type LogMiddleware struct {
	logger *zap.Logger
}

func Connect(args LoggerConnectProps) *LogMiddleware {

	var logger *zap.Logger

	if args.Production == true {
		logger = zap.New(otelzap.NewOtelCore(args.LoggerProvider))
		zap.ReplaceGlobals(logger)
		logger.Info("[Logger] Starting Logger with Prod Config")
	} else {
		logger, _ = zap.NewDevelopment()
	}

	return &LogMiddleware{logger: logger}
}

func (l *LogMiddleware) Logger(ctx context.Context) *zap.Logger {

	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return l.logger
	}

	return l.logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	)
}
