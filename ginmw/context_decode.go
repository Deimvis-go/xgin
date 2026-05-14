package ginmw

import (
	"github.com/gin-gonic/gin"
	"github.com/Deimvis-go/xgin/ginctx"
)

func AddContextDecodeCb(cb ginctx.DecodeCallbackFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		ginctx.AddDecodeCallback(c, cb)
		c.Next()
	}
}
