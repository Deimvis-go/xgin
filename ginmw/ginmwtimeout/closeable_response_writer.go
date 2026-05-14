package ginmwtimeout

import (
	"fmt"
	"net/http"
	"sync"
)

type closeableWriter struct {
	w http.ResponseWriter

	mu        sync.Mutex
	closedErr error
}

var _ http.ResponseWriter = (*closeableWriter)(nil)
var _ http.Pusher = (*closeableWriter)(nil)

func (cw *closeableWriter) Header() http.Header {
	return cw.w.Header()
}

func (cw *closeableWriter) Write(p []byte) (int, error) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	if cw.closedErr != nil {
		return 0, cw.closedErr
	}
	return cw.w.Write(p)
}

func (cw *closeableWriter) WriteHeader(statusCode int) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	if cw.closedErr != nil {
		return
	}
	cw.w.WriteHeader(statusCode)
}

func (cw *closeableWriter) Close(cause error) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.closeLocked(cause)
}

func (cw *closeableWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := cw.w.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (cw *closeableWriter) closeLocked(cause error) {
	if cw.closedErr != nil {
		return
	}
	cw.closedErr = fmt.Errorf("closed: %w", cause)
}
