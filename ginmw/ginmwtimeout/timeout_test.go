package ginmwtimeout

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
	"github.com/Deimvis/go-ext/go1.25/xptr"
	"github.com/Deimvis/models/utility/go/dmutil"
)

func TestMain(m *testing.M) {
	ginWriter := gin.DefaultWriter
	gin.DefaultWriter = io.Discard

	code := m.Run()

	gin.DefaultWriter = ginWriter

	os.Exit(code)
}

func TestTimeout_NotifyHandler(t *testing.T) {
	cfg := timeoutConfigSample
	cfg.DefaultDeadlineExpirationPolicy = &DeadlineExpirationPolicy{
		NotifyHandler: NotifyHandlerAction{
			Option: dmutil.Option{Enabled: xptr.T(true)},
		},
	}
	cfgTimeout1ms := cfg
	cfgTimeout1ms.DefaultTimeoutMs = 1
	t.Run("deadline-met/200", func(t *testing.T) {
		w := invokeHandler(cfg, func(c *gin.Context) {
			c.Writer.WriteHeader(200)
			_, err := c.Writer.Write([]byte("ok"))
			require.NoError(t, err)
		})
		require.Equal(t, 200, w.Code)
	})
	t.Run("deadline-met/post-write-ok", func(t *testing.T) {
		mw := func(c *gin.Context) {
			c.Next()
			_, err := c.Writer.Write([]byte("some extra data"))
			require.NoError(t, err)
		}
		w := invokeHandler(cfg, func(c *gin.Context) {
			c.Writer.WriteHeader(200)
			_, err := c.Writer.Write([]byte("ok"))
			require.NoError(t, err)
		}, mw)
		require.Equal(t, 200, w.Code)
	})
	t.Run("deadline-met/decoded-context-has-timeout", func(t *testing.T) {
		w := invokeHandler(cfg, func(c *gin.Context) {
			ctx := xmust.Do(ginctx.Decode(c))
			actDedl, ok := ctx.Deadline()
			require.True(t, ok)
			startTime, ok := GetRequestStartTime(c)
			require.True(t, ok)
			timeout, ok := GetRequestTimeout(c)
			require.True(t, ok)
			require.Equal(t, startTime.Add(timeout), actDedl)
			c.Writer.WriteHeader(200)
			_, err := c.Writer.Write([]byte("ok"))
			require.NoError(t, err)
		})
		require.Equal(t, 200, w.Code)
	})
	t.Run("deadline-met/panic/error-propagated", func(t *testing.T) {
		expErr := errors.New("my error")
		mw := func(c *gin.Context) {
			defer func() {
				r := recover()
				require.NotNil(t, r)
				require.ErrorIs(t, r.(error), expErr)
			}()
			c.Next()
		}
		w := invokeHandler(cfg, func(c *gin.Context) {
			panic(expErr)
		}, mw)
		require.Equal(t, 200, w.Code)
	})
	t.Run("deadline-expired/post-write-ok", func(t *testing.T) {
		mw := func(c *gin.Context) {
			c.Next()
			_, err := c.Writer.Write([]byte("some extra data"))
			require.NoError(t, err)
		}
		w := invokeHandler(cfgTimeout1ms, func(c *gin.Context) {
			<-xmust.Do(ginctx.Decode(c)).Done()
			c.String(408, "deadline expired")
		}, mw)
		require.Equal(t, 408, w.Code)
	})
	t.Run("deadline-expired/panic/error-propagated", func(t *testing.T) {
		expErr := errors.New("my error")
		mw := func(c *gin.Context) {
			defer func() {
				r := recover()
				require.NotNil(t, r)
				require.ErrorIs(t, r.(error), expErr)
			}()
			c.Next()
		}
		_ = invokeHandler(cfgTimeout1ms, func(c *gin.Context) {
			<-xmust.Do(ginctx.Decode(c)).Done()
			panic(expErr)
		}, mw)
	})
}

// NOTE: tests are disabled, because synctest.Test
// is introduced in go1.25, but CI uses go1.24.
// Anyway this test suite would be skipped.
//
// func TestTimeout_CloseResponse(t *testing.T) {
// 	t.Skip("gin implementation is not possible yet")
// 	cfg := timeoutConfigSample
// 	cfg.DefaultDeadlineExpirationPolicy = &DeadlineExpirationPolicy{
// 		CloseResponse: CloseResponseAction{
// 			Option:                     dmutil.Option{Enabled: xptr.T(true)},
// 			OverwriteToTimeoutResponse: true,
// 		},
// 	}

// 	startPostWrite := make(chan int, 1)
// 	finishPostWrite := make(chan int, 1)
// 	setupPostWrite := func() func() {
// 		startPostWrite = make(chan int, 1)
// 		finishPostWrite = make(chan int, 1)
// 		return func() {
// 			close(startPostWrite)
// 			close(finishPostWrite)
// 		}
// 	}
// 	myErr := errors.New("my error")

// 	h200ok := func(c *gin.Context, advance <-chan int) {
// 		<-advance
// 		c.Writer.WriteHeader(200)
// 		c.Writer.Write([]byte("ok"))
// 	}
// 	h200ok_noerr := func(c *gin.Context, advance <-chan int) {
// 		<-advance
// 		c.Writer.WriteHeader(200)
// 		_, err := c.Writer.Write([]byte("ok"))
// 		require.NoError(t, err)
// 	}
// 	h200ok_err := func(c *gin.Context, advance <-chan int) {
// 		<-advance
// 		c.Writer.WriteHeader(200)
// 		_, err := c.Writer.Write([]byte("ok"))
// 		require.Error(t, err)
// 	}

// 	tcs := []struct {
// 		title    string
// 		hPayload func(c *gin.Context, advanceHandler <-chan int)
// 		check    func(t *testing.T, h gin.HandlerFunc, advanceHandler chan<- int, advanceTimeout chan<- int)
// 	}{
// 		{
// 			"deadline-met/200",
// 			h200ok_noerr,
// 			func(t *testing.T, h gin.HandlerFunc, advanceHandler chan<- int, advanceTimeout chan<- int) {
// 				advanceHandler <- 1
// 				w := invokeHandler(cfg, h)
// 				require.Equal(t, 200, w.Code)
// 				require.Equal(t, "ok", w.Body.String())
// 			},
// 		},
// 		{
// 			"deadline-met/post-write-error",
// 			func(c *gin.Context, advance <-chan int) {
// 				<-advance
// 				c.Writer.WriteHeader(200)
// 				_, err := c.Writer.Write([]byte("ok"))
// 				require.NoError(t, err)
// 				go func() {
// 					<-startPostWrite
// 					_, err := c.Writer.Write([]byte("some extra data"))
// 					require.Error(t, err)
// 					finishPostWrite <- 1
// 				}()
// 			},
// 			func(t *testing.T, h gin.HandlerFunc, advanceHandler chan<- int, advanceTimeout chan<- int) {
// 				clean := setupPostWrite()
// 				defer clean()

// 				advanceHandler <- 1
// 				w := invokeHandler(cfg, h)
// 				require.Equal(t, 200, w.Code)
// 				require.Equal(t, "ok", w.Body.String())
// 				startPostWrite <- 1
// 				<-finishPostWrite
// 				require.Equal(t, 200, w.Code)
// 				require.Equal(t, "ok", w.Body.String())
// 			},
// 		},
// 		{
// 			"deadline-met/panic/error-propagated",
// 			func(c *gin.Context, advance <-chan int) {
// 				<-advance
// 				panic(myErr)
// 			},
// 			func(t *testing.T, h gin.HandlerFunc, advanceHandler, advanceTimeout chan<- int) {
// 				mw := func(c *gin.Context) {
// 					defer func() {
// 						r := recover()
// 						require.NotNil(t, r)
// 						require.ErrorIs(t, r.(error), myErr)
// 					}()
// 					c.Next()
// 				}
// 				advanceHandler <- 1
// 				_ = invokeHandler(cfg, h, mw)
// 			},
// 		},
// 		{
// 			"deadline-expired/408",
// 			h200ok,
// 			func(t *testing.T, h gin.HandlerFunc, advanceHandler, advanceTimeout chan<- int) {
// 				advanceTimeout <- 1
// 				w := invokeHandler(cfg, h)
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())
// 				advanceHandler <- 1
// 				synctest.Wait()
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())
// 			},
// 		},
// 		{
// 			"deadline-expired/pre-write-overriden",
// 			func(c *gin.Context, advanceHandler <-chan int) {
// 				c.Writer.WriteHeader(200)
// 				_, err := c.Writer.Write([]byte("ok"))
// 				require.NoError(t, err)
// 				// hack
// 				advanceTimeout := xmust.Ok(c.Get("advance-timeout")).(chan<- int)
// 				advanceTimeout <- 1
// 				<-advanceHandler
// 			},
// 			func(t *testing.T, h gin.HandlerFunc, advanceHandler, advanceTimeout chan<- int) {
// 				hackedH := func(c *gin.Context) {
// 					// hack
// 					c.Set("advance-timeout", advanceTimeout)
// 					h(c)
// 				}
// 				w := invokeHandler(cfg, hackedH)
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())
// 				advanceHandler <- 1
// 				synctest.Wait()
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())
// 			},
// 		},
// 		{
// 			"deadline-expired/post-write-error",
// 			h200ok_err,
// 			func(t *testing.T, h gin.HandlerFunc, advanceHandler, advanceTimeout chan<- int) {
// 				advanceTimeout <- 1
// 				w := invokeHandler(cfg, h)
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())
// 				advanceHandler <- 1
// 				synctest.Wait()
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())
// 			},
// 		},
// 		{
// 			"deadline-expired/concurrent-write",
// 			func(c *gin.Context, advance <-chan int) {
// 				c.Writer.WriteHeader(200)
// 				for {
// 					select {
// 					case <-advance:
// 						return
// 					case <-time.After(time.Second):
// 						c.Writer.Write([]byte("ok"))
// 					}
// 				}
// 			},
// 			func(t *testing.T, h gin.HandlerFunc, advanceHandler, advanceTimeout chan<- int) {
// 				clean := setupPostWrite()
// 				defer clean()

// 				var w *httptest.ResponseRecorder
// 				doWriteTimeRate := time.Second // just random time
// 				go func() {
// 					w = invokeHandler(cfg, h)
// 				}()
// 				synctest.Wait()

// 				for range 100 {
// 					// we're in the bubble,
// 					// so Sleep just advances time
// 					// and doesn't really blocks
// 					time.Sleep(doWriteTimeRate)
// 				}
// 				advanceTimeout <- 1
// 				synctest.Wait()
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())

// 				for range 100 {
// 					time.Sleep(doWriteTimeRate)
// 				}
// 				advanceHandler <- 1
// 				synctest.Wait()
// 				require.Equal(t, 408, w.Code)
// 				require.Equal(t, "deadline expired", w.Body.String())
// 			},
// 		},
// 		// TODO: "deadline-expired/panic/error-ignored"
// 		// TODO: "deadline-expired/408/repeated-on-same-engine"
// 		// TODO: "deadline-expired/post-write/repeated-on-same-engine"
// 	}
// 	for _, tc := range tcs {
// 		t.Run(tc.title, func(t *testing.T) {
// 			advanceHandler := make(chan int, 1)
// 			advanceTimeout := make(chan int, 1)

// 			mwtest.TimeAfter = func(d time.Duration) <-chan time.Time {
// 				ch := make(chan time.Time, 1)
// 				go func() {
// 					<-advanceTimeout
// 					ch <- time.Now()
// 					close(ch)
// 				}()
// 				return ch
// 			}
// 			defer func() {
// 				mwtest.TimeAfter = time.After
// 			}()
// 			h := func(c *gin.Context) {
// 				tc.hPayload(c, advanceHandler)
// 			}

// 			synctest.Test(t, func(t *testing.T) {
// 				defer func() {
// 					close(advanceHandler)
// 					close(advanceTimeout)
// 					synctest.Wait()
// 				}()
// 				tc.check(t, h, advanceHandler, advanceTimeout)
// 			})
// 		})
// 	}
// }

func TestTimeout_NotifyHandler_CtxValue_AccessFromHandler(t *testing.T) {
	tcs := []struct {
		title string
		fn    func(t *testing.T, c *gin.Context)
	}{
		{
			"directly",
			func(t *testing.T, c *gin.Context) {
				timeout, ok := GetRequestTimeout(c)
				require.True(t, ok)
				require.Greater(t, timeout, time.Duration(0))
			},
		},
		{
			"in_nested_function",
			func(t *testing.T, c *gin.Context) {
				myfn := func(ctx context.Context) {
					timeout, ok := GetRequestTimeout(ctx)
					require.True(t, ok)
					require.Greater(t, timeout, time.Duration(0))
				}
				myfn(c)
			},
		},
		{
			"after_gin_context_copy",
			func(t *testing.T, c *gin.Context) {
				c = c.Copy()
				timeout, ok := GetRequestTimeout(c)
				require.True(t, ok)
				require.Greater(t, timeout, time.Duration(0))
			},
		},
		{
			"after_context_copy",
			func(t *testing.T, c *gin.Context) {
				type key struct{}
				var ctx context.Context = c
				ctx = context.WithValue(ctx, key{}, "value")
				timeout, ok := GetRequestTimeout(ctx)
				require.True(t, ok)
				require.Greater(t, timeout, time.Duration(0))
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.title, func(t *testing.T) {
			r := gin.New()
			r.Use(Timeout(&timeoutConfigSample, zap.S()))
			r.GET("/", func(c *gin.Context) {
				tc.fn(t, c)
				c.Writer.WriteHeader(200)
				_, err := c.Writer.Write([]byte("ok"))
				require.NoError(t, err)
			})

			require200(t, invokeEndpoint(r, "GET", "/"))
		})
	}
}

func TestTimeout_NotifyHandler_CtxValue_AccessFromHanlder_NoMiddleware(t *testing.T) {
	testCases := []struct {
		title string
		fn    func(t *testing.T, c *gin.Context)
	}{
		{
			"directly",
			func(t *testing.T, c *gin.Context) {
				timeout, ok := GetRequestTimeout(c)
				require.False(t, ok)
				require.Equal(t, timeout, time.Duration(0))
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) {
				tc.fn(t, c)
				c.Writer.WriteHeader(200)
				_, err := c.Writer.Write([]byte("ok"))
				require.NoError(t, err)
			})

			require200(t, invokeEndpoint(r, "GET", "/"))
		})
	}
}

func TestTimeout_RegexpRules(t *testing.T) {
	defaultTimeoutMs := 5000
	testCases := []struct {
		title string
		rules []RegexpRule
		path  string
		exp   time.Duration
	}{
		{
			"default",
			timeoutConfigSample.RegexpRules,
			"/",
			5000 * time.Millisecond,
		},
		{
			"1000ms",
			timeoutConfigSample.RegexpRules,
			"/timeout1000ms",
			1000 * time.Millisecond,
		},
		{
			"9000ms",
			timeoutConfigSample.RegexpRules,
			"/timeout9000ms",
			9000 * time.Millisecond,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cfg := MiddlewareConfig{
				DefaultTimeoutMs: defaultTimeoutMs,
				RegexpRules:      tc.rules,
			}
			r := gin.New()
			r.Use(Timeout(&cfg, zap.S()))
			r.Use(func(c *gin.Context) {
				timeout, ok := GetRequestTimeout(c)
				require.True(t, ok)
				require.Equal(t, tc.exp, timeout)
			})
			r.GET("/", func(c *gin.Context) {
				c.Writer.WriteHeader(200)
				_, err := c.Writer.Write([]byte("ok"))
				require.NoError(t, err)
			})
			r.GET("/timeout1000ms", func(c *gin.Context) {
				c.Writer.WriteHeader(200)
				_, err := c.Writer.Write([]byte("ok"))
				require.NoError(t, err)
			})
			r.GET("/timeout9000ms", func(c *gin.Context) {
				c.Writer.WriteHeader(200)
				_, err := c.Writer.Write([]byte("ok"))
				require.NoError(t, err)
			})

			require200(t, invokeEndpoint(r, "GET", tc.path))
		})
	}
}

var timeoutConfigSample = MiddlewareConfig{
	DefaultTimeoutMs: 5000,
	RegexpRules: []RegexpRule{
		{
			PathRegexp: "/timeout1000ms",
			TimeoutMs:  1000,
		},
		{
			PathRegexp: "/timeout9000ms",
			TimeoutMs:  9000,
		},
	},
}

func invokeHandler(cfg MiddlewareConfig, h func(c *gin.Context), parentMws ...gin.HandlerFunc) *httptest.ResponseRecorder {
	r := gin.New()
	r.Use(parentMws...)
	r.Use(Timeout(&cfg, zap.S()))
	r.GET("/", h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	r.ServeHTTP(w, req)
	return w
}

func invokeEndpoint(h http.Handler, method string, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), method, path, nil)
	h.ServeHTTP(w, req)
	return w
}

func require200(t *testing.T, w *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	require.Equal(t, 200, w.Code)
	return w
}
