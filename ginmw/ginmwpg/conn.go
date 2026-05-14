package ginmwpg

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/Deimvis-go/xpg/pg"
	"github.com/Deimvis-go/xpg/pg/pgconn"
	"github.com/Deimvis-go/xpg/pg/pgtrace"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
	"github.com/Deimvis/go-ext/go1.25/xoptional"
)

// TODO: implement LazyConn(mode, cp, opts...); for ConnBuilder: LazyRO(), LazyRW()

func Conn(mode pg.ConnMode, cp pgconn.Provider, opts ...pgconn.AcquireOption) gin.HandlerFunc {
	return func(c *gin.Context) {
		release := acquireConn(c, cp, mode, opts...)
		defer release()
		c.Next()
	}
}

func NewConnBuilder() ConnBuilder {
	return &connBuilder{}
}

type ConnBuilder interface {
	WithConnProvider(cp pgconn.Provider) ConnBuilder
	WithOpts(opts ...pgconn.AcquireOption) ConnBuilder
	ClearOpts() ConnBuilder
	WithMode(m pgconn.Mode) ConnBuilder
	Build() gin.HandlerFunc

	// RO is a shortcut for WithMode(pgconn.RO).Build()
	RO() gin.HandlerFunc
	// RW is a shortcut for WithMode(pgconn.RW).Build()
	RW() gin.HandlerFunc
}

type connBuilder struct {
	cp   pgconn.Provider
	opts []pgconn.AcquireOption
	mode pgconn.Mode
}

func (cb *connBuilder) WithConnProvider(cp pgconn.Provider) ConnBuilder {
	cb.cp = cp
	return cb
}

func (cb *connBuilder) WithOpts(opts ...pgconn.AcquireOption) ConnBuilder {
	cb.opts = append(cb.opts, opts...)
	return cb
}

func (cb *connBuilder) ClearOpts() ConnBuilder {
	cb.opts = nil
	return cb
}

func (cb *connBuilder) WithMode(m pgconn.Mode) ConnBuilder {
	cb.mode = m
	return cb
}

func (cb *connBuilder) Build() gin.HandlerFunc {
	xmust.NotNilInterface(cb.cp, "conn provider not set")
	xmust.True(cb.mode == pgconn.RO || cb.mode == pgconn.RW, "conn mode not set")
	return Conn(cb.mode, cb.cp, cb.opts...)
}

func (cb *connBuilder) RO() gin.HandlerFunc {
	return cb.WithMode(pgconn.RO).Build()
}

func (cb *connBuilder) RW() gin.HandlerFunc {
	return cb.WithMode(pgconn.RW).Build()
}

func acquireConn(c *gin.Context, cp pgconn.Provider, mode pg.ConnMode, opts ...pgconn.AcquireOption) func() {
	// decode context for safety, because we may leak it into goroutine,
	// also automatically applies callbacks from different middlewares
	// (e.g. may add logging "req_id" field)
	ctx := xmust.Do(ginctx.Decode(c))
	opts = append(opts, pgconn.AcquireWithMeta(pgtrace.ConnAcquireMeta{
		ConnMode: xoptional.New(mode),
	}))
	conn, own, err := cp.Acquire(ctx, mode, opts...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = newAcquireConnTimeoutError(mode)
		}
		panic(err)
	}
	free := func(context.Context) error { return nil }
	if own.HasValue() {
		free = own.Value().MustTake().FreeConn
	}
	connCtxKey := pg.CtxConnKey(mode)
	c.Set(connCtxKey, conn)
	return func() {
		free(context.WithoutCancel(ctx))
		c.Set(connCtxKey, nil)
	}
}

func newAcquireConnTimeoutError(mode pg.ConnMode) AcquireConnTimeoutError {
	return AcquireConnTimeoutError{mode: mode}
}

type AcquireConnTimeoutError struct {
	mode pg.ConnMode
}

func (e AcquireConnTimeoutError) Error() string {
	return fmt.Sprintf("acquire conn timeout (mode=%s)", e.mode)
}

func (e AcquireConnTimeoutError) Unwrap() error {
	return context.DeadlineExceeded
}
