package jog

import (
	"math/rand"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ProdLogging(logLevel string) (Logging, error) {
	zapLogger, err := ProdLogger(logLevel)
	return Logging{zapLogger}, err
}

func ProdLogger(logLevel string) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if logLevel != "" {
		if err := (&level).UnmarshalText([]byte(logLevel)); err != nil {
			return nil, err
		}
	}

	rand.Seed(int64(time.Now().Nanosecond()))
	logConfig := zap.NewProductionConfig()
	logConfig.Sampling = nil
	logConfig.Level = zap.NewAtomicLevelAt(level)
	return logConfig.Build()
}

func DevLogging(logLevel string) (Logging, error) {
	zapLogger, err := DevLogger(logLevel)
	return Logging{zapLogger}, err
}

func DevLogger(logLevel string) (*zap.Logger, error) {
	rand.Seed(int64(time.Now().Nanosecond()))
	return zap.NewDevelopment(
		zap.AddStacktrace(zapcore.FatalLevel),
		zap.AddCallerSkip(1),
	)
}
