package services

import "go.uber.org/zap"

// ZapLoggerAdapter adapts a zap.Logger to the interfaces.Logger contract.
// It keeps logging lightweight by using the sugared logger for flexible fields.
type ZapLoggerAdapter struct {
	base *zap.Logger
}

// NewZapLoggerAdapter creates a new adapter from a zap logger.
func NewZapLoggerAdapter(logger *zap.Logger) *ZapLoggerAdapter {
	return &ZapLoggerAdapter{base: logger}
}

// Debug logs a debug message.
func (z *ZapLoggerAdapter) Debug(msg string, fields ...interface{}) {
	if z == nil || z.base == nil {
		return
	}
	z.base.Sugar().Debugw(msg, fields...)
}

// Info logs an info message.
func (z *ZapLoggerAdapter) Info(msg string, fields ...interface{}) {
	if z == nil || z.base == nil {
		return
	}
	z.base.Sugar().Infow(msg, fields...)
}

// Warn logs a warning message.
func (z *ZapLoggerAdapter) Warn(msg string, fields ...interface{}) {
	if z == nil || z.base == nil {
		return
	}
	z.base.Sugar().Warnw(msg, fields...)
}

// Error logs an error message.
func (z *ZapLoggerAdapter) Error(msg string, fields ...interface{}) {
	if z == nil || z.base == nil {
		return
	}
	z.base.Sugar().Errorw(msg, fields...)
}
