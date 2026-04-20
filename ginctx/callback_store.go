package ginctx

import (
	"sync"
	"sync/atomic"
)

func newCallbackStore[CallbackT any]() *callbackStore[CallbackT] {
	return &callbackStore[CallbackT]{}
}

type callbackStore[CallbackT any] struct {
	m       sync.Map
	anonCnt atomic.Int64
}

func (cs *callbackStore[CallbackT]) AddAnonymousCallback(cb CallbackT) {
	v := cs.anonCnt.Add(1) - 1
	cs.m.Store(v, cb)
}

func (cs *callbackStore[CallbackT]) Range(fn func(name any, cb CallbackT) bool) {
	cs.m.Range(func(k, v any) bool {
		return fn(k, v.(CallbackT))
	})
}
