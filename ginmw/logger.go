package ginmw

import (
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/Deimvis-go/ms/ms/msconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Logger(logger *zap.SugaredLogger) gin.HandlerFunc {
	return ginzap.GinzapWithConfig(logger.Desugar(), &ginzap.Config{
		UTC:          false,
		TimeFormat:   time.RFC3339,
		DefaultLevel: zapcore.InfoLevel,
		Context: ginzap.Fn(func(c *gin.Context) []zapcore.Field {
			reqId := GetRequestIdOr(c, "unknown")

			clientCloudServiceVs := c.Request.Header[msconfig.XClientCloudService]
			clientCloudService := "unknown"
			if len(clientCloudServiceVs) > 0 {
				clientCloudService = clientCloudServiceVs[0]
			}

			return []zapcore.Field{zap.String("req_id", reqId), zap.String("client_cloud_service", clientCloudService)}
		}),
	})
}
