// Package ginmwctx stores values set by ginmw middlewares inside gin/Go
// contexts. It is internal: import via the parent ginmw package.
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

const requestIdCtxKey = "request_id__github.com/Deimvis-go/xgin/ginmw"
