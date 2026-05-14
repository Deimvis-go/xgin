package ginmwup

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/Deimvis-go/xgin/ginmw"
	"go.uber.org/zap"
)

func Verbose(mw gin.HandlerFunc, mwName string, lg *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqId := ginmw.GetRequestIdOr(c, "unknown")
		lg.Debugw(fmt.Sprintf("Middleware %s {", mwName), "req_id", reqId)
		defer lg.Debugw(fmt.Sprintf("Middleware %s }", mwName), "req_id", reqId)
		mw(c)
	}
}
