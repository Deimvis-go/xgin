package ginctx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	t.Run("no-cb/proxy-kvs", func(t *testing.T) {
		c := &gin.Context{}
		c.Set("a", 1)
		c.Set("b", 2)
		ctx, err := Decode(c)
		require.NoError(t, err)
		require.Equal(t, 1, ctx.Value("a"))
		require.Equal(t, 2, ctx.Value("b"))
	})
	t.Run("single-cb/applied", func(t *testing.T) {
		c := &gin.Context{}
		c.Set("a", 1)
		c.Set("b", 2)
		AddDecodeCallback(c, func(c *gin.Context, dst context.Context) (context.Context, error) {
			return context.WithValue(dst, "c", 3), nil
		})
		ctx, err := Decode(c)
		require.NoError(t, err)
		require.Equal(t, 3, ctx.Value("c"))
	})
	t.Run("single-cb/proxy-kvs", func(t *testing.T) {
		c := &gin.Context{}
		c.Set("a", 1)
		c.Set("b", 2)
		AddDecodeCallback(c, func(c *gin.Context, dst context.Context) (context.Context, error) {
			return context.WithValue(dst, "c", 3), nil
		})
		ctx, err := Decode(c)
		require.NoError(t, err)
		require.Equal(t, 1, ctx.Value("a"))
		require.Equal(t, 2, ctx.Value("b"))
	})
	t.Run("single-cb/overwrite", func(t *testing.T) {
		c := &gin.Context{}
		c.Set("a", 1)
		c.Set("b", 2)
		AddDecodeCallback(c, func(c *gin.Context, dst context.Context) (context.Context, error) {
			return context.WithValue(dst, "a", 100), nil
		})
		ctx, err := Decode(c)
		require.NoError(t, err)
		require.Equal(t, 100, ctx.Value("a"))
	})
	t.Run("multiple-cb/applied", func(t *testing.T) {
		c := &gin.Context{}
		for i := range 100 {
			AddDecodeCallback(c, func(c *gin.Context, dst context.Context) (context.Context, error) {
				return context.WithValue(dst, i, i+1), nil
			})
		}
		ctx, err := Decode(c)
		require.NoError(t, err)
		for i := range 100 {
			require.Equal(t, i+1, ctx.Value(i))
		}
	})
	t.Run("proxy-deadline-depending-on-gin", func(t *testing.T) {
		test := func(t *testing.T, ctxFallback bool) {
			reqCtx := context.Background()
			dedl := time.Now().Add(time.Second)
			reqCtx, cancel := context.WithDeadline(reqCtx, dedl)
			defer cancel()
			req, err := http.NewRequestWithContext(reqCtx, "GET", "/", nil)
			require.NoError(t, err)

			r := gin.New()
			r.ContextWithFallback = ctxFallback
			r.GET("/", func(c *gin.Context) {
				ctx, err := Decode(c)
				require.NoError(t, err)
				actDedl, ok := ctx.Deadline()
				if ctxFallback {
					require.True(t, ok)
					require.Equal(t, dedl, actDedl)
				} else {
					require.False(t, ok)
				}
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, 200, w.Code)
		}
		t.Run("ctxfallback-on", func(t *testing.T) {
			test(t, true)
		})
		t.Run("ctxfallback-off", func(t *testing.T) {
			test(t, false)
		})
	})
}

func TestDecodeKey(t *testing.T) {
	t.Run("single-key", func(t *testing.T) {
		type a struct{}
		c := &gin.Context{}
		c.Set("a", 1)
		AddDecodeCallback(c, ShouldDecodeKey("a", a{}))
		ctx, err := Decode(c)
		require.NoError(t, err)
		require.Nil(t, ctx.Value("a"))
		require.Equal(t, 1, ctx.Value(a{}))
	})
	t.Run("should", func(t *testing.T) {
		type a struct{}
		c := &gin.Context{}
		c.Set("b", 2)
		AddDecodeCallback(c, ShouldDecodeKey("a", a{}))
		ctx, err := Decode(c)
		require.Error(t, err)
		require.Nil(t, ctx)
	})
	t.Run("may", func(t *testing.T) {
		type a struct{}
		c := &gin.Context{}
		c.Set("b", 2)
		AddDecodeCallback(c, MayDecodeKey("a", a{}))
		ctx, err := Decode(c)
		require.NoError(t, err)
		require.Nil(t, ctx.Value("a"))
		require.Nil(t, ctx.Value(a{}))
		require.Equal(t, 2, ctx.Value("b"))
	})
}
