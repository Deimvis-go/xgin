package ginmwtimeout

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Deimvis-go/logs/logs"
	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/Deimvis-go/xgin/ginmw/internal/ginmwctx"
	"github.com/Deimvis-go/xgin/ginmw/internal/mwtest"
	"github.com/Deimvis/go-ext/go1.25/ext"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
	"github.com/Deimvis/go-ext/go1.25/xptr"
	"github.com/Deimvis/models/utility/go/dmutil"
)

func Timeout(cfg *MiddlewareConfig, lg *zap.SugaredLogger) gin.HandlerFunc {
	// TODO: communication between middlewares - check that timeout middleware is first, otherwise - warning (?)

	// TODO: make exp policy required (legacy)
	if cfg.DefaultDeadlineExpirationPolicy == nil {
		cfg.DefaultDeadlineExpirationPolicy = &DeadlineExpirationPolicy{
			NotifyHandler: NotifyHandlerAction{Option: dmutil.Option{Enabled: xptr.T(true)}},
			CloseResponse: CloseResponseAction{Option: dmutil.Option{Enabled: xptr.T(false)}, OverwriteToTimeoutResponse: true},
		}
	}
	if cfg.DefaultDeadlineExpirationPolicy.CloseResponse.IsEnabled() {
		panic(errors.New("gin implementation is not possible yet"))
	}

	rules := ext.Map(cfg.RegexpRules, func(r RegexpRule) timeoutMWRule {
		re := xmust.Do(regexp.Compile(r.PathRegexp))
		t := time.Duration(r.TimeoutMs) * time.Millisecond
		return timeoutMWRule{Regexp: re, Timeout: t}
	})
	defaultTimeout := time.Duration(cfg.DefaultTimeoutMs) * time.Millisecond
	defaultExpPolicy := *cfg.DefaultDeadlineExpirationPolicy
	tc := &timeoutController{
		bufPool: sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
		lg: logs.ZapAsKVCtxLogger(lg),
	}
	mw := func(c *gin.Context) {
		reqId := ginmwctx.GetRequestIdOr(c, "unknown")
		path := c.Request.URL.Path
		for _, rule := range rules {
			if rule.Regexp.Match([]byte(path)) {
				lg.Debugw("Request timeout is matched by regexp", "req_id", reqId, "path", path, "regexp", rule.Regexp, "timeout", rule.Timeout)
				tc.handle(c, rule.Timeout, defaultExpPolicy)
				return
			}
		}
		lg.Debugw("Request timeout is default", "req_id", reqId, "path", path, "timeout", defaultTimeout)
		tc.handle(c, defaultTimeout, defaultExpPolicy)
	}
	return mw
}

type timeoutController struct {
	bufPool sync.Pool
	lg      logs.KVCtxLogger
}

func (tc *timeoutController) handle(c *gin.Context, t time.Duration, exp DeadlineExpirationPolicy) {
	// TODO: add decode cb that applies deadline to decoded one if it hasn't pass deadline

	startTime := time.Now()
	dedl := startTime.Add(t)

	ginmwctx.SetRequestStartTime(c, startTime)
	ginmwctx.SetRequestTimeout(c, t)

	if exp.NotifyHandler.IsEnabled() {
		cancels := []context.CancelFunc{}
		defer func() {
			for _, c := range cancels {
				c()
			}
		}()

		newReqContext, cancel := context.WithDeadline(c.Request.Context(), dedl)
		cancels = append(cancels, cancel)
		c.Request = c.Request.WithContext(newReqContext)

		// depending on gin engine configuration
		// it may not proxy context deadline from request's context,
		// so we add decode callback in order to make sure
		// that decoded gin context has deadline from timeout middleware.
		ginctx.AddDecodeCallback(c, func(c *gin.Context, dst context.Context) (context.Context, error) {
			var cancel context.CancelFunc
			dst, cancel = context.WithDeadline(dst, dedl)
			cancels = append(cancels, cancel)
			return dst, nil
		})
	}

	if exp.CloseResponse.IsEnabled() {
		// NOTE: currently, no valid implementation exists
		// for Gin framework since any implementation
		// leaves an opportunity for dangling handler to spoil
		// its context that was already transferred to another
		// request, because of gin's context pooling
		// (even though, we are allowed to override
		// gin.Context.Writer, we still has no ability
		// to forbid handler call Next(), which is automatically
		// continued once handler returns).
		// Other frameworks face similar issues,
		// but "echo" managed to workaround them
		// with http.TimeoutHandler given that
		// echo.Context doesn't store call stack state
		// and does not require looking after
		// Next() calls.
		// Native implementation for Gin using timeoutWriter
		// implemented in timeout_response_writer.go
		// will be possible when gin will:
		// 1. Make gin.Context an interface;
		// 2. Add access to all internals of gin.Context;
		// 3. Add an ability to prevent Next()
		// being called implicitly after return;
		// First and second points would allow to implement
		// making a standalone closeable copy of gin.Context
		// (which would be great if gin.Context natively supported):
		// - this copy would survive gin's context pooling
		// and allow to "close" context (forbidding all "write"-like
		// operations), so underlying handler would fail to interact
		// with response content, when timeout middleware already
		// sent a timeout response, "closed" the response
		// and returned context to the gin pool.
		panic("gin implementation is not possible yet")
		// tc._closeResponse_nativeImpl(c, dedl)
	} else {
		c.Next()
	}
}

func (tc *timeoutController) _closeResponse_nativeImpl(c *gin.Context, dedl time.Time) {
	// --------
	// In case all requirements are met,
	// here is a native implementation of timeout middleware.
	// Just in case, here is list of issues
	// that would happen if using this implementation
	// with gin<=1.11.0:
	// - if deadline expires and middleware exits,
	// before main goroutine reached end handler,
	// gin will implicitly call Next() on current process,
	// and everything will become a mess;
	// - if deadline expires and middleware exits,
	// it would return gin.Context to gin engine's pool,
	// and dangling handler would have an ability
	// to interact with context that was already reused
	// for another (new) request.
	// - if any middleware is placed before this one,
	// it's post-writes would cause an error and won't be applied.
	// --------
	origW := c.Writer
	buf := tc.bufPool.Get().(*bytes.Buffer)
	defer tc.bufPool.Put(buf)
	buf.Reset()
	tw := newTimeoutWriter(
		origW,
		buf,
		func(msg string) {
			tc.lg.Warn(context.Background(), msg)
		},
	)
	defer tw.Close(errors.New("unexpected cause"))
	c.Writer = tw
	// We cannot defer setting c.Writer = origW
	// because in case operation is timed out
	// we cannot allow handler to write response
	// after we finished the response.
	// Since timeout case is possible
	// and there is no common approach to communicate
	// between middlewares,
	// we cannot allow parent middlewares (if any)
	// to write response after timeout middleware ends,
	// therefore in any possible case, we close the writer,
	// so implementation is always valid
	// (always ready for timeout case).

	handlerDone := make(chan struct{})
	handlerPanicked := make(chan any, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				handlerPanicked <- p
			}
		}()
		c.Next()
		close(handlerDone)
	}()

	select {
	case p := <-handlerPanicked:
		tw.Close(errors.New("handler panicked"))
		panic(p)
	case <-handlerDone:
		// It is not valid to write
		// response after handler returned,
		// but we still obtain lock for safety.
		func() {
			tw.mu.Lock()
			defer tw.mu.Unlock()

			maps.Copy(origW.Header(), tw.Header())
			code := tw.Status()
			if code != 0 {
				origW.WriteHeader(code)
			}
			body := tw.Bytes()
			if len(body) > 0 {
				origW.Write(body)
			}

			tw.closeLocked(errors.New("handler done"))
		}()
	case <-mwtest.TimeAfter(time.Until(dedl)):
		tw.Close(errors.New("deadline expired"))
		origW.WriteHeader(408)
		// TODO: extract current process info and write here
		_, _ = io.WriteString(origW, "deadline expired")
		origW.Flush()
		// TODO: gin support: prevent Next() being called on this mw return
		// TODO: gin support: prevent dangling handler to interact with context
		// c.Abort()
	}
}

var GetRequestStartTime = ginmwctx.GetRequestStartTime
var GetRequestTimeout = ginmwctx.GetRequestTimeout

type timeoutMWRule struct {
	Regexp  *regexp.Regexp
	Timeout time.Duration
}
