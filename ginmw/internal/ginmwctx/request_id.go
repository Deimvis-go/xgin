package ginmwctx

import (
	"context"

	"github.com/gin-gonic/gin"
)

func GetRequestId(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIdCtxKey).(string)
	return v, ok
}

func GetRequestIdOr(ctx context.Context, fallback string) string {
	v, ok := ctx.Value(requestIdCtxKey).(string)
	if !ok {
		return fallback
	}
	return v
}

func SetRequestId(c *gin.Context, v string) {
	c.Set(requestIdCtxKey, v)
}

var requestIdCtxKey = "request_id__github.com/Deimvis-go/xgin/ginmw"
