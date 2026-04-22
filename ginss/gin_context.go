package ginss

import (
	"context"
	"errors"

	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
	"github.com/gin-gonic/gin"
)

// RetrieveGinContext allows to retrieve *gin.Context back from context.Context.
// NOTE: one should avoid using this function as much as possible,
// it should only be used when there's no other way to achieve the result.
func RetrieveGinContext(ctx context.Context) (*gin.Context, error) {
	c := ctx.Value(ginContextCtxKey{})
	if c == nil {
		return nil, errNoGinContext
	}
	return c.(*gin.Context), nil
}

func MustRetrieveGinContext(ctx context.Context) *gin.Context {
	return xmust.Do(RetrieveGinContext(ctx))
}

func ConcealGinContext(ctx context.Context, c *gin.Context) context.Context {
	// TODO: wait until godzen Go is 1.24 or higher and use weak pointer
	// in order to help garbage collector remove this context later
	// (circular reference: gin context inside of gin context)
	return context.WithValue(ctx, ginContextCtxKey{}, c)
}

type ginContextCtxKey struct{}

var errNoGinContext = errors.New("no gin context (make sure ginss.NewHandler was used)")
