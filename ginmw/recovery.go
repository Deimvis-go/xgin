package ginmw

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandlerFunc handles a specific recovered error.
type ErrorHandlerFunc = func(c *gin.Context, err error)

// Recovery returns a middleware that recovers panics and logs them through lg.
//
// customErrorHandlers maps a target error value to a handler. A recovered
// error is matched against a target using [errors.As] semantics; if a match
// is found, the corresponding handler is invoked instead of the default
// behavior (which responds with HTTP 500).
//
// Panics caused by non-error values are responded to with an empty JSON body
// and HTTP 500.
//
// Any target key in customErrorHandlers must implement [error]; otherwise
// Recovery panics at construction time.
func Recovery(lg *zap.SugaredLogger, customErrorHandlers map[any]ErrorHandlerFunc) gin.HandlerFunc {
	validateCustomErrorHandlers(customErrorHandlers)
	var infoPool sync.Pool
	infoPool.New = func() any {
		return &recoveryInfo{
			stackBuf: make([]byte, stackBufInitSize),
		}
	}
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			info := infoPool.Get().(*recoveryInfo)
			defer infoPool.Put(info)
			info.SetStack()
			reqId := GetRequestIdOr(c, "unknown")
			req := c.Request

			switch rr := r.(type) {
			case error:
				err := rr
				lg.Errorw("Recovered on error", "err", err.Error(),
					"req_id", reqId,
					"req_method", req.Method,
					"req_path", req.URL.Path,
					"stack", string(info.stackBuf[:info.stackSize]),
				)
				for targetErr, customHandler := range customErrorHandlers {
					t, ok := matchError(err, targetErr)
					if ok {
						customHandler(c, t.(error))
						return
					}
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			default:
				lg.Errorw("Recovered on unknown entity", "entity", rr,
					"req_id", reqId,
					"req_method", req.Method,
					"req_path", req.URL.Path,
					"stack", string(info.stackBuf[:info.stackSize]),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{})
			}
		}()
		c.Next()
	}
}

// matchError is like [errors.As] but does not mutate target; it allocates a
// new value of target's type and returns it on match.
func matchError(err error, target any) (any, bool) {
	newT := reflect.New(reflect.TypeOf(target))
	t := newT.Interface()
	if errors.As(err, t) {
		return newT.Elem().Interface(), true
	}
	return nil, false
}

func validateCustomErrorHandlers(customErrorHandlers map[any]ErrorHandlerFunc) {
	errorT := reflect.TypeOf((*error)(nil)).Elem()
	for targetErr := range customErrorHandlers {
		targetErrT := reflect.TypeOf(targetErr)
		if !targetErrT.Implements(errorT) {
			panic(fmt.Errorf("target error (%s) does not implement error interface", targetErrT.Name()))
		}
	}
}

type recoveryInfo struct {
	stackBuf  []byte
	stackSize int
}

func (rec *recoveryInfo) SetStack() {
	for {
		n := runtime.Stack(rec.stackBuf, false)
		if n < len(rec.stackBuf) {
			rec.stackSize = n
			return
		}
		rec.stackBuf = make([]byte, 2*len(rec.stackBuf))
	}
}

const stackBufInitSize = 1024
