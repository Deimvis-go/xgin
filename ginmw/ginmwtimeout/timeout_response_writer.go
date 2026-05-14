package ginmwtimeout

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NOTE: implementation is inspired by http.TimeoutHandler

func newTimeoutWriter(w gin.ResponseWriter, buf *bytes.Buffer, warnHandler func(msg string)) *timeoutWriter {
	return &timeoutWriter{
		closeableWriter: &closeableWriter{
			w: &buffWriter{
				buf:         buf,
				code:        0,
				header:      make(http.Header),
				warnHandler: warnHandler,
			},
		},
		origW: w,
	}
}

type timeoutWriter struct {
	*closeableWriter
	origW gin.ResponseWriter
}

var _ http.ResponseWriter = (*timeoutWriter)(nil)
var _ http.Hijacker = (*timeoutWriter)(nil)
var _ http.Flusher = (*timeoutWriter)(nil)
var _ http.CloseNotifier = (*timeoutWriter)(nil)
var _ gin.ResponseWriter = (*timeoutWriter)(nil)

// NOTE for gin.ResponseWriter impl:
// - http.Hijacker is banned (error)
// - http.Flusher is ignored
// - http.CloseNotifier is proxied to original writer
// - Pusher() is proxied to original writer
// - WriteHeaderNow() is ignored
// - other gin methods return information about current
// buffered response writer (not original one)

func (tw *timeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("timeout writer does not allow to hijack connection")
}

func (tw *timeoutWriter) Flush() {
}

func (tw *timeoutWriter) CloseNotify() <-chan bool {
	return tw.origW.CloseNotify()
}

func (tw *timeoutWriter) Pusher() http.Pusher {
	return tw.origW.Pusher()
}

func (tw *timeoutWriter) Status() int {
	bw := tw.buffWriter()
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.code
}

func (tw *timeoutWriter) Size() int {
	bw := tw.buffWriter()
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.buf.Len()
}

func (tw *timeoutWriter) WriteString(s string) (int, error) {
	bw := tw.buffWriter()
	bw.mu.Lock()
	return io.WriteString(bw, s)
}

func (tw *timeoutWriter) Written() bool {
	bw := tw.buffWriter()
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.code != 0 || bw.buf.Len() > 0
}

func (tw *timeoutWriter) WriteHeaderNow() {
	return
}

func (tw *timeoutWriter) Bytes() []byte {
	bw := tw.buffWriter()
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.buf.Bytes()
}

func (tw *timeoutWriter) buffWriter() *buffWriter {
	return tw.closeableWriter.w.(*buffWriter)
}
