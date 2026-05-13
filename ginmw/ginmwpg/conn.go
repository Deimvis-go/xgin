package ginmwpg

import (
	"context"
	"errors"
	"fmt"

	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
	"github.com/gin-gonic/gin"
)

// Conn returns a middleware that acquires a connection from cp in the given
// mode, stores it in the gin context under [CtxConnKey](mode), and releases
// it when the request finishes.
//
// If the acquire fails because the request context's deadline expired, the
// middleware panics with an [AcquireConnTimeoutError]; other errors are
// re-panicked as-is so callers can plug in their own recovery middleware.
func Conn(mode Mode, cp Provider, opts ...AcquireOption) gin.HandlerFunc {
	return func(c *gin.Context) {
		release := acquireConn(c, cp, mode, opts...)
		defer release()
		c.Next()
	}
}

// NewConnBuilder returns a fluent builder over [Conn].
func NewConnBuilder() ConnBuilder {
	return &connBuilder{}
}

// ConnBuilder builds a [Conn] middleware step by step. It exists so callers
// can share a partially configured builder (provider, options) and finalize
// it per-route with [ConnBuilder.RO] or [ConnBuilder.RW].
type ConnBuilder interface {
	WithProvider(cp Provider) ConnBuilder
	WithOpts(opts ...AcquireOption) ConnBuilder
	ClearOpts() ConnBuilder
	WithMode(m Mode) ConnBuilder
	Build() gin.HandlerFunc

	// RO is a shortcut for WithMode(RO).Build().
	RO() gin.HandlerFunc
	// RW is a shortcut for WithMode(RW).Build().
	RW() gin.HandlerFunc
}

type connBuilder struct {
	cp   Provider
	opts []AcquireOption
	mode Mode
}

func (cb *connBuilder) WithProvider(cp Provider) ConnBuilder {
	cb.cp = cp
	return cb
}

func (cb *connBuilder) WithOpts(opts ...AcquireOption) ConnBuilder {
	cb.opts = append(cb.opts, opts...)
	return cb
}

func (cb *connBuilder) ClearOpts() ConnBuilder {
	cb.opts = nil
	return cb
}

func (cb *connBuilder) WithMode(m Mode) ConnBuilder {
	cb.mode = m
	return cb
}

func (cb *connBuilder) Build() gin.HandlerFunc {
	xmust.NotNilInterface(cb.cp, "conn provider not set")
	xmust.True(cb.mode == RO || cb.mode == RW, "conn mode not set")
	return Conn(cb.mode, cb.cp, cb.opts...)
}

func (cb *connBuilder) RO() gin.HandlerFunc {
	return cb.WithMode(RO).Build()
}

func (cb *connBuilder) RW() gin.HandlerFunc {
	return cb.WithMode(RW).Build()
}

func acquireConn(c *gin.Context, cp Provider, mode Mode, opts ...AcquireOption) func() {
	// Decode the gin context so we have a leak-safe context that survives
	// past c.Next(). Decode also applies callbacks installed by other
	// middlewares (e.g. ginmwtimeout adds the request deadline).
	ctx := xmust.Do(ginctx.Decode(c))
	conn, own, err := cp.Acquire(ctx, mode, opts...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = AcquireConnTimeoutError{mode: mode}
		}
		panic(err)
	}
	free := func(context.Context) error { return nil }
	if own.HasValue() {
		free = own.Value().MustTake().FreeConn
	}
	key := ctxConnKey(mode)
	c.Set(key, conn)
	return func() {
		free(context.WithoutCancel(ctx))
		c.Set(key, nil)
	}
}

// AcquireConnTimeoutError is returned when [Provider.Acquire] fails because
// the request's deadline expired. It unwraps to [context.DeadlineExceeded].
type AcquireConnTimeoutError struct {
	mode Mode
}

func (e AcquireConnTimeoutError) Error() string {
	return fmt.Sprintf("acquire conn timeout (mode=%s)", e.mode)
}

func (e AcquireConnTimeoutError) Unwrap() error {
	return context.DeadlineExceeded
}
