package ginmwtimeout

// func Timeout(cfg *MiddlewareConfig, lg *zap.SugaredLogger) gin.HandlerFunc {
// 	newTimeoutMW := func(t time.Duration) gin.HandlerFunc {
// 		h := timeout.New(
// 			timeout.WithTimeout(t),
// 			timeout.WithHandler(func(c *gin.Context) {}),
// 			timeout.WithResponse(func(c *gin.Context) {
// 				c.String(408, "timeout")
// 			}),
// 		)
// 		return func(c *gin.Context) {
// 			ginmwctx.SetRequestStartTime(c, time.Now())
// 			ginmwctx.SetRequestTimeout(c, t)
// 			h(c)
// 		}
// 	}
// 	rules := ext.Map(cfg.RegexpRules, func(r RegexpRule) timeoutMWRule {
// 		re := xmust.Do(regexp.Compile(r.PathRegexp))
// 		t := time.Duration(r.TimeoutMs) * time.Millisecond
// 		return timeoutMWRule{Regexp: re, Timeout: t, TimeoutMW: newTimeoutMW(t)}
// 	})
// 	defaultTimeout := time.Duration(cfg.DefaultTimeoutMs) * time.Millisecond
// 	defaultTimeoutMW := newTimeoutMW(defaultTimeout)
// 	mw := func(c *gin.Context) {
// 		reqId := ginmwctx.GetRequestIdOr(c, "unknown")
// 		path := c.Request.URL.Path
// 		for _, rule := range rules {
// 			if rule.Regexp.Match([]byte(path)) {
// 				lg.Debugw("Request timeout is matched by regexp", "req_id", reqId, "path", path, "regexp", rule.Regexp, "timeout", rule.Timeout)
// 				rule.TimeoutMW(c)
// 				return
// 			}
// 		}
// 		lg.Debugw("Request timeout is default", "req_id", reqId, "path", path, "timeout", defaultTimeout)
// 		defaultTimeoutMW(c)
// 	}
// 	return mw
// }

// var GetRequestStartTime = ginmwctx.GetRequestStartTime
// var GetRequestTimeout = ginmwctx.GetRequestTimeout

// type timeoutMWRule struct {
// 	Regexp    *regexp.Regexp
// 	Timeout   time.Duration
// 	TimeoutMW func(c *gin.Context)
// }
