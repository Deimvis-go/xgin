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

	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func ptr[T any](v T) *T { return &v }

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
		NotifyHandler: NotifyHandlerAction{Enabled: ptr(true)},
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
			ctx, err := ginctx.Decode(c)
			require.NoError(t, err)
			actDedl, ok := ctx.Deadline()
			require.True(t, ok)
			startTime, ok := GetRequestStartTime(c)
			require.True(t, ok)
			timeout, ok := GetRequestTimeout(c)
			require.True(t, ok)
			require.Equal(t, startTime.Add(timeout), actDedl)
			c.Writer.WriteHeader(200)
			_, err = c.Writer.Write([]byte("ok"))
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
			ctx, err := ginctx.Decode(c)
			require.NoError(t, err)
			<-ctx.Done()
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
			ctx, err := ginctx.Decode(c)
			require.NoError(t, err)
			<-ctx.Done()
			panic(expErr)
		}, mw)
	})
}

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
		{PathRegexp: "/timeout1000ms", TimeoutMs: 1000},
		{PathRegexp: "/timeout9000ms", TimeoutMs: 9000},
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
