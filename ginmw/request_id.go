package ginmw

import (
	"fmt"

	"github.com/Deimvis-go/xgin/ginmw/internal/ginmwctx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestId returns a middleware that assigns a request id to every incoming
// request, picked from a configured request header or generated afresh.
// The request id can be retrieved from the request's context via
// [GetRequestId] / [GetRequestIdOr].
func RequestId(cfg *RequestIdMiddlewareConfig) gin.HandlerFunc {
	if err := cfg.validate(); err != nil {
		panic(fmt.Errorf("ginmw: invalid RequestId config: %w", err))
	}
	return func(c *gin.Context) {
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
		if requestId == "" {
			panic("ginmw: generated request id is empty")
		}

		ginmwctx.SetRequestId(c, requestId)
		if cfg.Context != nil {
			for _, key := range cfg.Context.ExtraKeys {
				c.Set(key, requestId)
			}
		}

		if cfg.ResponseHeaders != nil {
			for _, name := range cfg.ResponseHeaders.HeaderNames {
				c.Writer.Header().Set(name, requestId)
			}
			if cfg.ResponseHeaders.ProxyMatchedRequestHeader != nil && *cfg.ResponseHeaders.ProxyMatchedRequestHeader {
				if header != nil {
					c.Writer.Header().Set(header.name, requestId)
				}
			}
		}

		c.Next()
	}
}

// GetRequestId returns the request id previously set by the [RequestId]
// middleware.
var GetRequestId = ginmwctx.GetRequestId

// GetRequestIdOr returns the request id, or fallback if none was set.
var GetRequestIdOr = ginmwctx.GetRequestIdOr

func findRequestIdHeader(cfg *RequestIdRequestHeadersConfig, c *gin.Context) *headerEntry {
	switch cfg.ResolutionAlgorithm {
	case HRA_FirstNotEmpty:
		for name, values := range c.Request.Header {
			if containsStr(cfg.CandidateHeaderNames, name) && len(values) > 0 && len(values[0]) > 0 {
				return &headerEntry{name: name, value: values[0]}
			}
		}
	default:
		panic(fmt.Errorf("ginmw: unknown resolution algorithm: %s", cfg.ResolutionAlgorithm))
	}
	return nil
}

func genRequestId(cfg *RequetsIdGenerationConfig) string {
	switch cfg.Algorithm {
	case RIDGA_UUID4:
		return uuid.New().String()
	default:
		panic(fmt.Errorf("ginmw: unknown request id generation algorithm: %s", cfg.Algorithm))
	}
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

type headerEntry struct {
	name  string
	value string
}

// DefaultRequestIdHeaderName is the header name used by [DefaultRequestIdConfig].
var DefaultRequestIdHeaderName = "X-Request-Id"

// DefaultRequestIdConfig is a reasonable default configuration for the
// [RequestId] middleware: it reads "X-Request-Id" on incoming requests,
// writes it on the response, and generates a UUIDv4 when absent.
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
