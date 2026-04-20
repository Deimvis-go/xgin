package ginmw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRecovery(t *testing.T) {
	t.Run("no-panic/no-logs", func(t *testing.T) {
		w, logs := invokeRecoveryHandler(func(c *gin.Context) {
			c.Writer.WriteHeader(200)
			_, err := c.Writer.Write([]byte("ok"))
			require.NoError(t, err)
		})
		require.Equal(t, 200, w.Code)
		require.Len(t, logs, 0)
	})
	t.Run("panic/recovered", func(t *testing.T) {
		require.NotPanics(t, func() {
			w, _ := invokeRecoveryHandler(func(c *gin.Context) {
				panic(123)
			})
			require.Equal(t, 500, w.Code)
		})
	})
	t.Run("panic/logs", func(t *testing.T) {
		w, logs := invokeRecoveryHandler(func(c *gin.Context) {
			panic(123)
		})
		require.Equal(t, 500, w.Code)
		require.Len(t, logs, 1)
		require.Equal(t, "Recovered on unknown entity", logs[0].Message)
	})
}

func TestRecovery_MatchError(t *testing.T) {
	tcs := []struct {
		title         string
		e             TestError
		target        TestError
		exp           bool
		expSameTarget bool
	}{
		{"not-matched", err1A, err2A, false, false},
		{"top-level-exact-match", err1A, err1A, true, true},
		{"top-level-type-match", err1A, err1B, true, false},
		{"top-level-exact-match_having-internal", err2A_1A, err2A, true, false},
		{"internal-level-exact-match", err2A_1A, err1A, true, true},
		{"internal-level-type-match", err2A_1B, err1A, true, false},
	}

	for _, tc := range tcs {
		t.Run(tc.title, func(t *testing.T) {
			res, ok := matchError(tc.e, tc.target)
			require.Equal(t, tc.exp, ok)
			if ok {
				require.Equal(t, reflect.TypeOf(tc.target), reflect.TypeOf(res))
			}
			if tc.expSameTarget {
				require.Equal(t, tc.target, res)
			} else {
				require.NotEqual(t, tc.target, res)
			}
		})
	}
}

func invokeRecoveryHandler(h func(c *gin.Context), parentMws ...gin.HandlerFunc) (*httptest.ResponseRecorder, []observer.LoggedEntry) {
	core, recorded := observer.New(zapcore.InfoLevel)
	lg := zap.New(core, zap.AddCaller()).Sugar()

	r := gin.New()
	r.Use(parentMws...)
	r.Use(Recovery(lg, nil))
	r.GET("/", h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	r.ServeHTTP(w, req)
	return w, recorded.TakeAll()
}

type TestError interface {
	error
	Clone() TestError
	CmpEq(TestError) bool
}

var (
	err1A    = &testError1{"error 1A", nil}
	err1B    = &testError1{"error 1B", nil}
	err2A    = &testError2{"error 2A", nil}
	err2A_1A = &testError2{"error 2A", err1A}
	err2A_1B = &testError2{"error 2A", err1B}
)

type testError1 struct {
	msg      string
	internal error
}

func (e *testError1) Error() string     { return e.msg }
func (e *testError1) Unwrap() error     { return e.internal }
func (e *testError1) Clone() TestError  { c := *e; return &c }
func (e *testError1) CmpEq(o TestError) bool {
	if x, ok := o.(*testError1); ok {
		return e.msg == x.msg && e.internal == x.internal
	}
	return false
}

type testError2 struct {
	msg      string
	internal error
}

func (e *testError2) Error() string     { return e.msg }
func (e *testError2) Unwrap() error     { return e.internal }
func (e *testError2) Clone() TestError  { c := *e; return &c }
func (e *testError2) CmpEq(o TestError) bool {
	if x, ok := o.(*testError2); ok {
		return e.msg == x.msg && e.internal == x.internal
	}
	return false
}
