package ginctx

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
)

type DecodeCallbackFn func(c *gin.Context, dst context.Context) (context.Context, error)

func AddDecodeCallback(c *gin.Context, cb DecodeCallbackFn) {
	cbs, ok := getDecodeCallbacks(c)
	if !ok {
		cbs = newCallbackStore[DecodeCallbackFn]()
	}
	cbs.AddAnonymousCallback(cb)
	c.Set(decodeCbsCtxKey, cbs)
}

func ShouldDecodeKey(srcKey string, dstKey any) DecodeCallbackFn {
	return decodeKey(srcKey, dstKey)
}

func MayDecodeKey(srcKey string, dstKey any) DecodeCallbackFn {
	return func(c *gin.Context, dst context.Context) (context.Context, error) {
		ctx, err := decodeKey(srcKey, dstKey)(c, dst)
		if err != nil {
			return dst, nil
		}
		return ctx, err
	}
}

func decodeKey(srcKey string, dstKey any) DecodeCallbackFn {
	return func(c *gin.Context, dst context.Context) (context.Context, error) {
		if dstKeyStr, ok := dstKey.(string); ok && dstKeyStr == srcKey {
			return dst, nil
		}
		v, ok := c.Get(srcKey)
		if !ok {
			return nil, fmt.Errorf("no key in gin context: %s", srcKey)
		}
		dst = context.WithValue(dst, dstKey, v)
		dst = context.WithValue(dst, srcKey, nil)
		return dst, nil
	}
}

func Decode(c *gin.Context) (context.Context, error) {
	// Gin cleans underlying context after request processing,
	// and we can not guarantee that context won't be leaked
	// to background goroutine,
	// even std http library does that: https://github.com/gin-gonic/gin/issues/4117.
	// Note that we can't copy context keys solely,
	// since we can't obtain internal lock,
	// and end up with full copy returned by Copy method.
	var ctx context.Context = c.Copy()
	cbs, ok := getDecodeCallbacks(c)
	if ok {
		var err error
		cbs.Range(func(_ any, cb DecodeCallbackFn) bool {
			ctx, err = cb(c, ctx)
			return err == nil
		})
		if err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

func getDecodeCallbacks(c *gin.Context) (*callbackStore[DecodeCallbackFn], bool) {
	cbs_, ok := c.Get(decodeCbsCtxKey)
	if ok {
		cs, ok := cbs_.(*callbackStore[DecodeCallbackFn])
		xmust.True(ok)
		return cs, true
	}
	return nil, false
}

const decodeCbsCtxKey = "ginctx.decode_cbs__LwzbEncV6F3bzrtZAS29qk"
