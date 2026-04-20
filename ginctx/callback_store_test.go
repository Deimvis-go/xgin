package ginctx

import (
	"context"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCallbackStore(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		cs := newCallbackStore[DecodeCallbackFn]()
		cs.Range(func(name any, cb DecodeCallbackFn) bool {
			t.Fail()
			return false
		})
		require.Equal(t, int64(0), cs.anonCnt.Load())
	})
	t.Run("single-add", func(t *testing.T) {
		cs := newCallbackStore[DecodeCallbackFn]()
		cs.AddAnonymousCallback(func(c *gin.Context, dst context.Context) (context.Context, error) {
			return nil, nil
		})
		cnt := 0
		cs.Range(func(name any, cb DecodeCallbackFn) bool {
			cnt++
			return true
		})
		require.Equal(t, 1, cnt)
		require.Equal(t, int64(1), cs.anonCnt.Load())
	})
	t.Run("multiple-add/sequential", func(t *testing.T) {
		cs := newCallbackStore[DecodeCallbackFn]()
		for range 100 {
			cs.AddAnonymousCallback(func(c *gin.Context, dst context.Context) (context.Context, error) {
				return nil, nil
			})
		}
		cnt := 0
		cs.Range(func(name any, cb DecodeCallbackFn) bool {
			cnt++
			return true
		})
		require.Equal(t, 100, cnt)
		require.Equal(t, int64(100), cs.anonCnt.Load())
	})
	t.Run("multiple-add/concurrent", func(t *testing.T) {
		cs := newCallbackStore[DecodeCallbackFn]()
		wg := &sync.WaitGroup{}
		for range 100 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cs.AddAnonymousCallback(func(c *gin.Context, dst context.Context) (context.Context, error) {
					return nil, nil
				})
			}()
		}
		wg.Wait()
		cnt := 0
		cs.Range(func(name any, cb DecodeCallbackFn) bool {
			cnt++
			return true
		})
		require.Equal(t, 100, cnt)
		require.Equal(t, int64(100), cs.anonCnt.Load())
	})
}
