package ginmw

import (
	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/gin-gonic/gin"
)

// AddContextDecodeCb registers a decode callback for the current request.
// The callback is invoked by [ginctx.Decode].
func AddContextDecodeCb(cb ginctx.DecodeCallbackFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		ginctx.AddDecodeCallback(c, cb)
		c.Next()
	}
}
