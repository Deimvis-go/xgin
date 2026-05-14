package ginmwctx

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

func GetRequestStartTime(ctx context.Context) (time.Time, bool) {
	v, ok := ctx.Value(requestStartTimeCtxKey).(time.Time)
	return v, ok
}

func GetRequestTimeout(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(requestTimeoutCtxKey).(time.Duration)
	return v, ok
}

func GetCurrentOperation(ctx context.Context) (any, bool) {
	v := ctx.Value(requestCurrentOperationCtxKey)
	return v, v != nil
}

func SetRequestStartTime(c *gin.Context, v time.Time) {
	c.Set(requestStartTimeCtxKey, v)
}

func SetRequestTimeout(c *gin.Context, v time.Duration) {
	c.Set(requestTimeoutCtxKey, v)
}

var requestStartTimeCtxKey = "request_start_time__github.com/Deimvis-go/xgin/ginmw"
var requestTimeoutCtxKey = "request_timeout__github.com/Deimvis-go/xgin/ginmw"
var requestCurrentOperationCtxKey = "request_current_operation__github.com/Deimvis-go/xgin/ginmw"
