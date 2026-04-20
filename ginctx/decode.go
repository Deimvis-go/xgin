// Package ginctx provides tools to convert a [*gin.Context] into a
// leak-safe [context.Context] with support for pluggable decode callbacks.
package ginctx

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

// DecodeCallbackFn transforms dst by reading from the gin context.
// Returning a non-nil error aborts the decoding.
type DecodeCallbackFn func(c *gin.Context, dst context.Context) (context.Context, error)

// AddDecodeCallback registers a callback that runs during [Decode].
// Callbacks run in registration order and may modify the destination context.
func AddDecodeCallback(c *gin.Context, cb DecodeCallbackFn) {
	cbs, ok := getDecodeCallbacks(c)
	if !ok {
		cbs = newCallbackStore[DecodeCallbackFn]()
	}
	cbs.AddAnonymousCallback(cb)
	c.Set(decodeCbsCtxKey, cbs)
}

// ShouldDecodeKey returns a callback that moves a value from gin context key
// srcKey to destination context key dstKey. If srcKey is missing in gin
// context, the returned callback fails with an error.
func ShouldDecodeKey(srcKey string, dstKey any) DecodeCallbackFn {
	return decodeKey(srcKey, dstKey)
}

// MayDecodeKey is like [ShouldDecodeKey] but tolerates missing srcKey
// and returns the destination context unchanged in that case.
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

// Decode returns a [context.Context] derived from c that is safe to pass to
// background goroutines. All registered decode callbacks are applied in
// order; if any of them returns an error, Decode returns that error.
//
// Gin resets the underlying request context once the request finishes and
// cannot guarantee the context will not leak into a background goroutine
// (see https://github.com/gin-gonic/gin/issues/4117). Decode therefore
// returns a full copy via c.Copy instead of wrapping the live context.
func Decode(c *gin.Context) (context.Context, error) {
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
	v, ok := c.Get(decodeCbsCtxKey)
	if !ok {
		return nil, false
	}
	cs, ok := v.(*callbackStore[DecodeCallbackFn])
	if !ok {
		panic(fmt.Errorf("ginctx: unexpected type for %s: %T", decodeCbsCtxKey, v))
	}
	return cs, true
}

const decodeCbsCtxKey = "ginctx.decode_cbs__LwzbEncV6F3bzrtZAS29qk"
