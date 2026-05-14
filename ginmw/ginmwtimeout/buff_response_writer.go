package ginmwtimeout

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"runtime"
	"strings"
	"sync"
)

type buffWriter struct {
	buf         *bytes.Buffer
	code        int
	header      http.Header
	warnHandler func(msg string)

	mu        sync.Mutex
	wroteCode bool
}

var _ http.ResponseWriter = (*buffWriter)(nil)

func (bw *buffWriter) Header() http.Header {
	return bw.header
}

func (bw *buffWriter) Write(p []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.buf.Write(p)
}

func (bw *buffWriter) WriteHeader(code int) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.writeHeaderLocked(code)
}

func (bw *buffWriter) writeHeaderLocked(code int) {
	checkWriteHeaderCode(code)

	if bw.wroteCode {
		if bw.warnHandler != nil {
			caller := relevantCaller()
			msg := fmt.Sprintf("http: superfluous response.WriteHeader call from %s (%s:%d)", caller.Function, path.Base(caller.File), caller.Line)
			bw.warnHandler(msg)
		}
	} else {
		bw.wroteCode = true
		bw.code = code
	}
}

// --- source: net/http

func checkWriteHeaderCode(code int) {
	// Issue 22880: require valid WriteHeader status codes.
	// For now we only enforce that it's three digits.
	// In the future we might block things over 599 (600 and above aren't defined
	// at https://httpwg.org/specs/rfc7231.html#status.codes).
	// But for now any three digits.
	//
	// We used to send "HTTP/1.1 000 0" on the wire in responses but there's
	// no equivalent bogus thing we can realistically send in HTTP/2,
	// so we'll consistently panic instead and help people find their bugs
	// early. (We can't return an error from WriteHeader even if we wanted to.)
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", code))
	}
}

// relevantCaller searches the call stack for the first function outside of net/http.
// The purpose of this function is to provide more helpful error messages.
func relevantCaller() runtime.Frame {
	pc := make([]uintptr, 16)
	n := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:n])
	var frame runtime.Frame
	for {
		frame, more := frames.Next()
		if !strings.HasSuffix(packagePath(frame), "ginmw/ginmwtimeout") {
			return frame
		}
		if !more {
			break
		}
	}
	return frame
}

func packagePath(frame runtime.Frame) string {
	fnName := frame.Function
	slashInd := strings.LastIndexByte(fnName, '/')
	ind := strings.IndexByte(fnName[slashInd+1:], '.')
	if ind == -1 {
		return ""
	}
	return fnName[:slashInd+1+ind]
}
