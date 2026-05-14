package ginmw

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/Deimvis-go/valid"
	"github.com/Deimvis-go/xgin/ginmw/internal/ginmwctx"
	"github.com/Deimvis/go-ext/go1.25/ext"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xinvar"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
)

func RequestId(cfg *RequestIdMiddlewareConfig) gin.HandlerFunc {
	xmust.NoErr(valid.Deep(cfg))
	mw := func(c *gin.Context) {
		var requestId string
		var header *headerEntry
		if cfg.RequestHeaders != nil {
			header = findRequestIdHeader(cfg.RequestHeaders, c)
		}
		if header != nil {
			requestId = header.value
		} else {
			requestId = genRequestId(&cfg.IdGeneration)
		}
		xinvar.True(len(requestId) > 0)

		ginmwctx.SetRequestId(c, requestId)
		if cfg.Context != nil {
			for _, key := range cfg.Context.ExtraKeys {
				c.Set(key, requestId)
			}
		}

		// in gin header can't be written after body (http limitation, actually)
		if cfg.ResponseHeaders != nil {
			for _, name := range cfg.ResponseHeaders.HeaderNames {
				c.Writer.Header().Set(name, requestId)
			}
			if cfg.ResponseHeaders.ProxyMatchedRequestHeader != nil && *cfg.ResponseHeaders.ProxyMatchedRequestHeader {
				if header != nil {
					xinvar.Eq(header.value, requestId)
					c.Writer.Header().Set(header.name, requestId)
				}
			}
		}

		c.Next()
	}
	return mw
}

var GetRequestId = ginmwctx.GetRequestId
var GetRequestIdOr = ginmwctx.GetRequestIdOr

func findRequestIdHeader(cfg *RequestIdRequestHeadersConfig, c *gin.Context) *headerEntry {
	switch cfg.ResolutionAlgorithm {
	case HRA_FirstNotEmpty:
		for name, values := range c.Request.Header {
			if ext.Contains(cfg.CandidateHeaderNames, name) && len(values) > 0 && len(values[0]) > 0 {
				return &headerEntry{name: name, value: values[0]}
			}
		}
	default:
		panic(fmt.Errorf("got unexpected resolution algorithm: %s", cfg.ResolutionAlgorithm))
	}
	return nil
}

func genRequestId(cfg *RequetsIdGenerationConfig) string {
	var requestId string
	switch cfg.Algorithm {
	case RIDGA_UUID4:
		requestId = uuid.New().String()
	default:
		panic(fmt.Errorf("got unexpected request id generation algorithm: %s", cfg.Algorithm))
	}
	return requestId
}

type headerEntry struct {
	name  string
	value string
}

var DefaultRequestIdHeaderName = "X-Request-Id"

var DefaultRequestIdConfig = RequestIdMiddlewareConfig{
	RequestHeaders: &RequestIdRequestHeadersConfig{
		CandidateHeaderNames: []string{DefaultRequestIdHeaderName},
		ResolutionAlgorithm:  HRA_FirstNotEmpty,
	},
	ResponseHeaders: &RequestIdResponseHeadersConfig{
		HeaderNames: []string{DefaultRequestIdHeaderName},
	},
	IdGeneration: RequetsIdGenerationConfig{
		Algorithm: RIDGA_UUID4,
	},
}
