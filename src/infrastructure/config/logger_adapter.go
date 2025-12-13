package config

import (
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"go.uber.org/zap"
)

// ZapLoggerAdapter adapts zap.Logger to implement interfaces.Logger
type ZapLoggerAdapter struct {
	logger *zap.Logger
}

// NewZapLoggerAdapter creates a new adapter for zap.Logger
func NewZapLoggerAdapter(logger *zap.Logger) interfaces.Logger {
	return &ZapLoggerAdapter{
		logger: logger,
	}
}

// Debug logs a debug message
func (z *ZapLoggerAdapter) Debug(msg string, fields ...interface{}) {
	z.logger.Debug(msg, z.fieldsToZap(fields)...)
}

// Info logs an info message
func (z *ZapLoggerAdapter) Info(msg string, fields ...interface{}) {
	z.logger.Info(msg, z.fieldsToZap(fields)...)
}

// Warn logs a warning message
func (z *ZapLoggerAdapter) Warn(msg string, fields ...interface{}) {
	z.logger.Warn(msg, z.fieldsToZap(fields)...)
}

// Error logs an error message
func (z *ZapLoggerAdapter) Error(msg string, fields ...interface{}) {
	z.logger.Error(msg, z.fieldsToZap(fields)...)
}

// fieldsToZap converts interface{} fields to zap.Field
// This is a simplified implementation - in production, you'd want better handling
func (z *ZapLoggerAdapter) fieldsToZap(fields []interface{}) []zap.Field {
	var zapFields []zap.Field

	// Process pairs of key-value
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			value := fields[i+1]
			zapFields = append(zapFields, zap.Any(key, value))
		}
	}

	return zapFields
}
