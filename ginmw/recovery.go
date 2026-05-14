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

type ErrorHandlerFunc = func(c *gin.Context, err error)

// TODO: reimplement ginzap.CustomRecoveryWithZap in order to pass through request_id into log

// NOTE: for some reason errors.As accepts any as target (not only error interface),
// so we stick to the same approach and accept any as target error.
// --NOTE: actually, when `error` interface is passed as target, then any error will match
// and override target to itself (since any error implements error interface and so reflect.AssignableTo will pass).
func Recovery(lg *zap.SugaredLogger, customErrorHandlers map[any]ErrorHandlerFunc) gin.HandlerFunc {
	validateCustomErrorHandlers(customErrorHandlers)
	var infoPool sync.Pool
	infoPool.New = func() any {
		return &recoveryInfo{
			stackBuf: make([]byte, stackBufInitSize),
		}
	}
	mw := func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
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
					// NOTE: if you want to hide error response bodies, the right way is to do this on edge balancers
					//       or customize this middleware in order to support this, though I think it's unnecessary
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
			}
		}()
		c.Next()
	}
	return mw
}

// matchError works as errors.As, but does not override target.
func matchError(err error, target any) (any, bool) {
	newT := reflect.New(reflect.TypeOf(target))
	t := newT.Interface() // NOTE: we can't use x := newT.Elem().Interface(); errors.As(err, &x); since this will be a pointer to interface
	if errors.As(err, t) {
		// NOTE: we need to return value of the same type as input target as convention.
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
