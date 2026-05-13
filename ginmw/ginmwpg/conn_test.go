package ginmwpg_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/Deimvis-go/xgin/ginmw/ginmwpg"
	"github.com/Deimvis-go/xgin/ginmw/ginmwtimeout"
	"github.com/Deimvis/go-ext/go1.25/xoptional"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	ginWriter := gin.DefaultWriter
	gin.DefaultWriter = io.Discard
	gin.SetMode(gin.ReleaseMode)

	code := m.Run()

	gin.DefaultWriter = ginWriter
	os.Exit(code)
}

func TestConn(t *testing.T) {
	t.Run("conn-available", func(t *testing.T) {
		cp := &fakeProvider{conn: "my-conn"}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		var got any
		invokeHandler(func(c *gin.Context) {
			got = ginmwpg.CtxConn(c, ginmwpg.RO)
		}, mw)
		require.Equal(t, 1, cp.acquireCalls())
		require.Equal(t, "my-conn", got)
	})
	t.Run("conn-released", func(t *testing.T) {
		own := &fakeOwnership{}
		cp := &fakeProvider{conn: "my-conn", ownership: own}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		invokeHandler(func(c *gin.Context) {}, mw)
		require.Equal(t, 1, cp.acquireCalls())
		require.Equal(t, 1, own.takeCalls())
		require.Equal(t, 1, own.freeCalls())
	})
	t.Run("conn-cleared-after-request", func(t *testing.T) {
		own := &fakeOwnership{}
		cp := &fakeProvider{conn: "my-conn", ownership: own}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		var afterNextConn any
		seenConn := false
		outer := func(c *gin.Context) {
			c.Next()
			afterNextConn = ginmwpg.CtxConn(c, ginmwpg.RO)
		}
		invokeHandler(func(c *gin.Context) {
			seenConn = ginmwpg.CtxConn(c, ginmwpg.RO) != nil
		}, outer, mw)
		require.True(t, seenConn)
		require.Nil(t, afterNextConn)
	})
	t.Run("panic/conn-released", func(t *testing.T) {
		own := &fakeOwnership{}
		cp := &fakeProvider{conn: "my-conn", ownership: own}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		require.Panics(t, func() {
			invokeHandler(func(c *gin.Context) { panic(1) }, mw)
		})
		require.Equal(t, 1, own.freeCalls())
	})
	t.Run("acquire-deadline-exceeded/timeout-error", func(t *testing.T) {
		cp := &fakeProvider{err: context.DeadlineExceeded}
		mw := ginmwpg.Conn(ginmwpg.RW, cp)
		recovered := captureRecover(func() { invokeHandler(func(c *gin.Context) {}, mw) })
		require.NotNil(t, recovered)
		err, ok := recovered.(error)
		require.True(t, ok)
		var tErr ginmwpg.AcquireConnTimeoutError
		require.True(t, errors.As(err, &tErr))
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
	t.Run("acquire-error-propagated", func(t *testing.T) {
		want := errors.New("boom")
		cp := &fakeProvider{err: want}
		mw := ginmwpg.Conn(ginmwpg.RW, cp)
		recovered := captureRecover(func() { invokeHandler(func(c *gin.Context) {}, mw) })
		require.Same(t, want, recovered)
	})
	t.Run("opts-passed-through", func(t *testing.T) {
		optA := struct{ tag string }{tag: "a"}
		optB := struct{ tag string }{tag: "b"}
		cp := &fakeProvider{conn: "c"}
		mw := ginmwpg.Conn(ginmwpg.RO, cp, optA, optB)
		invokeHandler(func(c *gin.Context) {}, mw)
		require.Equal(t, []ginmwpg.AcquireOption{optA, optB}, cp.lastOpts())
	})
}

func TestConnBuilder(t *testing.T) {
	t.Run("RO-builds-RO-middleware", func(t *testing.T) {
		cp := &fakeProvider{conn: "ro-conn"}
		mw := ginmwpg.NewConnBuilder().WithProvider(cp).RO()
		var got any
		invokeHandler(func(c *gin.Context) {
			got = ginmwpg.CtxConn(c, ginmwpg.RO)
		}, mw)
		require.Equal(t, "ro-conn", got)
		require.Equal(t, ginmwpg.RO, cp.lastMode())
	})
	t.Run("RW-builds-RW-middleware", func(t *testing.T) {
		cp := &fakeProvider{conn: "rw-conn"}
		mw := ginmwpg.NewConnBuilder().WithProvider(cp).RW()
		var got any
		invokeHandler(func(c *gin.Context) {
			got = ginmwpg.CtxConn(c, ginmwpg.RW)
		}, mw)
		require.Equal(t, "rw-conn", got)
		require.Equal(t, ginmwpg.RW, cp.lastMode())
	})
	t.Run("missing-provider-panics", func(t *testing.T) {
		require.Panics(t, func() {
			ginmwpg.NewConnBuilder().WithMode(ginmwpg.RO).Build()
		})
	})
	t.Run("missing-mode-panics", func(t *testing.T) {
		cp := &fakeProvider{}
		require.Panics(t, func() {
			ginmwpg.NewConnBuilder().WithProvider(cp).Build()
		})
	})
	t.Run("ClearOpts-resets-opts", func(t *testing.T) {
		cp := &fakeProvider{}
		mw := ginmwpg.NewConnBuilder().
			WithProvider(cp).
			WithOpts("x").
			ClearOpts().
			RO()
		invokeHandler(func(c *gin.Context) {}, mw)
		require.Empty(t, cp.lastOpts())
	})
}

func TestConn_Integration_TimeoutMW(t *testing.T) {
	enabled := true
	cfg := ginmwtimeout.MiddlewareConfig{
		DefaultTimeoutMs: 5000,
		DefaultDeadlineExpirationPolicy: &ginmwtimeout.DeadlineExpirationPolicy{
			NotifyHandler: ginmwtimeout.NotifyHandlerAction{Enabled: &enabled},
		},
	}
	t.Run("deadline-propagates-to-acquire-context", func(t *testing.T) {
		cp := &fakeProvider{conn: "c"}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		invokeHandler(func(c *gin.Context) {}, ginmwtimeout.Timeout(&cfg, zap.S()), mw)
		require.Equal(t, 1, cp.acquireCalls())
		_, ok := cp.lastCtx().Deadline()
		require.True(t, ok, "acquire ctx should carry the timeout deadline")
	})
	t.Run("conn-released-on-handler-return", func(t *testing.T) {
		own := &fakeOwnership{}
		cp := &fakeProvider{conn: "c", ownership: own}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		invokeHandler(func(c *gin.Context) {}, ginmwtimeout.Timeout(&cfg, zap.S()), mw)
		require.Equal(t, 1, own.freeCalls())
	})
	t.Run("conn-released-on-panic", func(t *testing.T) {
		own := &fakeOwnership{}
		cp := &fakeProvider{conn: "c", ownership: own}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		require.Panics(t, func() {
			invokeHandler(func(c *gin.Context) { panic(1) }, ginmwtimeout.Timeout(&cfg, zap.S()), mw)
		})
		require.Equal(t, 1, own.freeCalls())
	})
	t.Run("expired-deadline/conn-released", func(t *testing.T) {
		own := &fakeOwnership{}
		cp := &fakeProvider{conn: "c", ownership: own}
		mw := ginmwpg.Conn(ginmwpg.RO, cp)
		expireMw := func(c *gin.Context) {
			expired, cancel := context.WithDeadline(c.Request.Context(), time.Unix(0, 0))
			defer cancel()
			c.Request = c.Request.WithContext(expired)
			c.Next()
		}
		invokeHandler(func(c *gin.Context) {
			ctx, err := ginctx.Decode(c)
			require.NoError(t, err)
			<-ctx.Done()
		}, ginmwtimeout.Timeout(&cfg, zap.S()), expireMw, mw)
		require.Equal(t, 1, own.freeCalls())
	})
}

func captureRecover(fn func()) (r any) {
	defer func() { r = recover() }()
	fn()
	return nil
}

func invokeHandler(h func(c *gin.Context), mws ...gin.HandlerFunc) *httptest.ResponseRecorder {
	r := gin.New()
	r.ContextWithFallback = true
	r.Use(mws...)
	r.GET("/", h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	r.ServeHTTP(w, req)
	return w
}

type fakeProvider struct {
	conn      any
	ownership ginmwpg.Ownership
	err       error

	acquireN  atomic.Int32
	lastModeV atomic.Value // Mode
	lastOptsV atomic.Value // []AcquireOption
	lastCtxV  atomic.Value // context.Context
}

func (p *fakeProvider) Acquire(ctx context.Context, mode ginmwpg.Mode, opts ...ginmwpg.AcquireOption) (any, xoptional.T[ginmwpg.Ownership], error) {
	p.acquireN.Add(1)
	p.lastModeV.Store(mode)
	p.lastCtxV.Store(ctx)
	if opts == nil {
		opts = []ginmwpg.AcquireOption{}
	}
	p.lastOptsV.Store(opts)
	if p.err != nil {
		return nil, xoptional.New[ginmwpg.Ownership](), p.err
	}
	if p.ownership != nil {
		return p.conn, xoptional.New(p.ownership), nil
	}
	return p.conn, xoptional.New[ginmwpg.Ownership](), nil
}

func (p *fakeProvider) acquireCalls() int          { return int(p.acquireN.Load()) }
func (p *fakeProvider) lastMode() ginmwpg.Mode     { return p.lastModeV.Load().(ginmwpg.Mode) }
func (p *fakeProvider) lastCtx() context.Context   { return p.lastCtxV.Load().(context.Context) }
func (p *fakeProvider) lastOpts() []ginmwpg.AcquireOption {
	v := p.lastOptsV.Load()
	if v == nil {
		return nil
	}
	return v.([]ginmwpg.AcquireOption)
}

type fakeOwnership struct {
	takeN atomic.Int32
	freeN atomic.Int32
}

func (o *fakeOwnership) MustTake() ginmwpg.OwnedConn {
	o.takeN.Add(1)
	return &fakeOwnedConn{parent: o}
}

func (o *fakeOwnership) takeCalls() int { return int(o.takeN.Load()) }
func (o *fakeOwnership) freeCalls() int { return int(o.freeN.Load()) }

type fakeOwnedConn struct {
	parent *fakeOwnership
}

func (c *fakeOwnedConn) FreeConn(context.Context) error {
	c.parent.freeN.Add(1)
	return nil
}
