// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jog

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logging is the default logging wrapper that can create
// logger instances either for a given Context or context-less.
type Logging struct {
	zapLogger *zap.Logger
}

// NewLogging creates a new Logging.
func NewLogging(logger *zap.Logger) Logging {
	return Logging{zapLogger: logger}
}

// IsNil checks if logging wraps a nil zapLogger
func (b Logging) IsNil() bool {
	return b.zapLogger == nil
}

// Bg creates a context-unaware logger.
func (b Logging) Bg() Logger {
	return logger(b)
}

// For returns a context-aware Logger. If the context
// contains an OpenTracing span, all logging calls are also
// echo-ed into the span.
func (b Logging) For(ctx context.Context) Logger {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		logger := spanLogger{span: span, logger: b.zapLogger}

		if jaegerCtx, ok := span.Context().(jaeger.SpanContext); ok {
			logger.spanFields = []zapcore.Field{
				zap.String("trace_id", jaegerCtx.TraceID().String()),
				zap.String("span_id", jaegerCtx.SpanID().String()),
			}
		}

		return logger
	}
	return b.Bg()
}

// With creates a child logging, and optionally adds some context fields to that logging.
func (b Logging) With(fields ...zapcore.Field) Logging {
	return Logging{zapLogger: b.zapLogger.With(fields...)}
}
