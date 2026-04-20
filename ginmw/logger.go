package ginmw

import (
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerConfig configures the [Logger] middleware.
type LoggerConfig struct {
	// TimeFormat used for the "time" field. Defaults to [time.RFC3339].
	TimeFormat string
	// UTC switches the "time" field to UTC.
	UTC bool
	// DefaultLevel is the log level for successful requests.
	// Defaults to [zapcore.InfoLevel].
	DefaultLevel zapcore.Level
	// ExtraFields lets callers inject additional log fields from a request
	// (e.g. selected headers). It is invoked on every access log entry.
	ExtraFields func(c *gin.Context) []zapcore.Field
}

// Logger returns a middleware that logs every request via the provided zap
// logger. The request id added by [RequestId] is attached as a "req_id" field.
func Logger(logger *zap.SugaredLogger, cfg LoggerConfig) gin.HandlerFunc {
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = time.RFC3339
	}
	if cfg.DefaultLevel == 0 {
		cfg.DefaultLevel = zapcore.InfoLevel
	}
	return ginzap.GinzapWithConfig(logger.Desugar(), &ginzap.Config{
		UTC:          cfg.UTC,
		TimeFormat:   cfg.TimeFormat,
		DefaultLevel: cfg.DefaultLevel,
		Context: ginzap.Fn(func(c *gin.Context) []zapcore.Field {
			fields := []zapcore.Field{zap.String("req_id", GetRequestIdOr(c, "unknown"))}
			if cfg.ExtraFields != nil {
				fields = append(fields, cfg.ExtraFields(c)...)
			}
			return fields
		}),
	})
}
