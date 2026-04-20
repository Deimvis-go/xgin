package ginmwtimeout

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/Deimvis-go/xgin/ginmw/internal/ginmwctx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Timeout returns a middleware that assigns a deadline to every incoming
// request.
//
// The deadline is computed from [MiddlewareConfig.DefaultTimeoutMs] and
// narrowed by the first matching [RegexpRule]. When the deadline expires,
// the behavior is governed by [MiddlewareConfig.DefaultDeadlineExpirationPolicy].
//
// Currently only the NotifyHandler policy is implemented — it cancels the
// request's context (and any context returned by [ginctx.Decode]) so that
// cooperating handlers can abort their work. CloseResponse is not supported
// on Gin yet.
func Timeout(cfg *MiddlewareConfig, lg *zap.SugaredLogger) gin.HandlerFunc {
	if cfg.DefaultDeadlineExpirationPolicy == nil {
		enabled := true
		disabled := false
		cfg.DefaultDeadlineExpirationPolicy = &DeadlineExpirationPolicy{
			NotifyHandler: NotifyHandlerAction{Enabled: &enabled},
			CloseResponse: CloseResponseAction{Enabled: &disabled, OverwriteToTimeoutResponse: true},
		}
	}
	if cfg.DefaultDeadlineExpirationPolicy.CloseResponse.IsEnabled() {
		panic(errors.New("ginmwtimeout: CloseResponse is not supported on Gin yet"))
	}

	rules := make([]timeoutMWRule, 0, len(cfg.RegexpRules))
	for _, r := range cfg.RegexpRules {
		re, err := regexp.Compile(r.PathRegexp)
		if err != nil {
			panic(fmt.Errorf("ginmwtimeout: invalid path regexp %q: %w", r.PathRegexp, err))
		}
		rules = append(rules, timeoutMWRule{Regexp: re, Timeout: time.Duration(r.TimeoutMs) * time.Millisecond})
	}
	defaultTimeout := time.Duration(cfg.DefaultTimeoutMs) * time.Millisecond
	defaultExpPolicy := *cfg.DefaultDeadlineExpirationPolicy

	return func(c *gin.Context) {
		reqId := ginmwctx.GetRequestIdOr(c, "unknown")
		path := c.Request.URL.Path
		for _, rule := range rules {
			if rule.Regexp.MatchString(path) {
				lg.Debugw("Request timeout is matched by regexp", "req_id", reqId, "path", path, "regexp", rule.Regexp, "timeout", rule.Timeout)
				handle(c, rule.Timeout, defaultExpPolicy)
				return
			}
		}
		lg.Debugw("Request timeout is default", "req_id", reqId, "path", path, "timeout", defaultTimeout)
		handle(c, defaultTimeout, defaultExpPolicy)
	}
}

func handle(c *gin.Context, t time.Duration, exp DeadlineExpirationPolicy) {
	startTime := time.Now()
	dedl := startTime.Add(t)

	ginmwctx.SetRequestStartTime(c, startTime)
	ginmwctx.SetRequestTimeout(c, t)

	if exp.NotifyHandler.IsEnabled() {
		var cancels []context.CancelFunc
		defer func() {
			for _, cancel := range cancels {
				cancel()
			}
		}()

		newReqContext, cancel := context.WithDeadline(c.Request.Context(), dedl)
		cancels = append(cancels, cancel)
		c.Request = c.Request.WithContext(newReqContext)

		// Depending on the gin engine configuration, the underlying handler
		// context may not inherit the deadline from the request context. Add
		// a decode callback so contexts produced by [ginctx.Decode] always
		// carry our deadline.
		ginctx.AddDecodeCallback(c, func(_ *gin.Context, dst context.Context) (context.Context, error) {
			var cancel context.CancelFunc
			dst, cancel = context.WithDeadline(dst, dedl)
			cancels = append(cancels, cancel)
			return dst, nil
		})
	}

	c.Next()
}

// GetRequestStartTime returns the time at which the [Timeout] middleware
// started processing the request.
var GetRequestStartTime = ginmwctx.GetRequestStartTime

// GetRequestTimeout returns the timeout chosen by the [Timeout] middleware
// for the current request.
var GetRequestTimeout = ginmwctx.GetRequestTimeout

type timeoutMWRule struct {
	Regexp  *regexp.Regexp
	Timeout time.Duration
}
