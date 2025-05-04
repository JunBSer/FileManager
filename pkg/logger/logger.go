package logger

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

const (
	Key       = "logger"
	RequestID = "requestID"
)

type Logger interface {
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Error(ctx context.Context, msg string, fields ...zap.Field)
	Debug(ctx context.Context, msg string, fields ...zap.Field)
	CreateChildLogger(fields ...zap.Field) Logger
}

type logger struct {
	log *zap.Logger
}

func New(serviceName, lvlInfo string) Logger {
	var zapLevel zapcore.Level

	config := zap.NewProductionConfig()

	switch lvlInfo {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	config.Level = zap.NewAtomicLevelAt(zapLevel)
	zapLogger, err := config.Build()

	if err != nil {
		log.Fatalln(err, serviceName)
	}

	return logger{log: zapLogger.With(zap.String("service", serviceName))}
}

func (l logger) Info(ctx context.Context, msg string, fields ...zap.Field) {

	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String("requestID", ctx.Value(RequestID).(string)))
	}

	l.log.Info(msg, fields...)
}

func (l logger) Error(ctx context.Context, msg string, fields ...zap.Field) {

	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String("requestID", ctx.Value(RequestID).(string)))
	}
	l.log.Error(msg, fields...)
}

func (l logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {

	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String("requestID", ctx.Value(RequestID).(string)))
	}

	l.log.Debug(msg, fields...)
}

func (l logger) CreateChildLogger(fields ...zap.Field) Logger {
	return &logger{l.log.With(fields...)}
}

func GetLoggerFromContext(ctx context.Context) Logger {
	return ctx.Value(Key).(Logger)
}
