package ginmwup

import (
	"fmt"

	"github.com/Deimvis-go/xgin/ginmw"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Verbose wraps a middleware with debug-level log lines that mark the entry
// and exit of the middleware. The request id (if present) is included.
func Verbose(mw gin.HandlerFunc, mwName string, lg *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqId := ginmw.GetRequestIdOr(c, "unknown")
		lg.Debugw(fmt.Sprintf("Middleware %s {", mwName), "req_id", reqId)
		defer lg.Debugw(fmt.Sprintf("Middleware %s }", mwName), "req_id", reqId)
		mw(c)
	}
}
