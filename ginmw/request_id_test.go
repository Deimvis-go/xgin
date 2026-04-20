package ginmw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestRequestId_AccessFromHanlder(t *testing.T) {
	testCases := []struct {
		title string
		fn    func(t *testing.T, c *gin.Context)
	}{
		{
			"directly",
			func(t *testing.T, c *gin.Context) {
				reqId, ok := GetRequestId(c)
				require.True(t, ok)
				require.NotEmpty(t, reqId)
				reqId = GetRequestIdOr(c, "")
				require.NotEmpty(t, reqId)
			},
		},
		{
			"in_nested_function",
			func(t *testing.T, c *gin.Context) {
				myfn := func(ctx context.Context) {
					reqId, ok := GetRequestId(ctx)
					require.True(t, ok)
					require.NotEmpty(t, reqId)
					reqId = GetRequestIdOr(ctx, "")
					require.NotEmpty(t, reqId)
				}
				myfn(c)
			},
		},
		{
			"after_gin_context_copy",
			func(t *testing.T, c *gin.Context) {
				c = c.Copy()
				reqId, ok := GetRequestId(c)
				require.True(t, ok)
				require.NotEmpty(t, reqId)
			},
		},
		{
			"after_context_copy",
			func(t *testing.T, c *gin.Context) {
				type key struct{}
				var ctx context.Context = c
				ctx = context.WithValue(ctx, key{}, "value")
				reqId, ok := GetRequestId(ctx)
				require.True(t, ok)
				require.NotEmpty(t, reqId)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			r := gin.New()
			r.Use(RequestId(&DefaultRequestIdConfig))
			r.GET("/", func(c *gin.Context) {
				tc.fn(t, c)
				c.String(200, "ok")
			})

			makeRequest(t, r)
		})
	}
}

func TestRequestId_AccessFromHanlder_NoMiddleware(t *testing.T) {
	testCases := []struct {
		title string
		fn    func(t *testing.T, c *gin.Context)
	}{
		{
			"directly",
			func(t *testing.T, c *gin.Context) {
				reqId, ok := GetRequestId(c)
				require.False(t, ok)
				require.Empty(t, reqId)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) {
				tc.fn(t, c)
				c.String(200, "ok")
			})

			makeRequest(t, r)
		})
	}
}

func TestRequestId_AccessFromResponse(t *testing.T) {
	testCases := []struct {
		title string
		rh    RequestIdResponseHeadersConfig
	}{
		{
			"default_config",
			*DefaultRequestIdConfig.ResponseHeaders,
		},
		{
			"single_header",
			RequestIdResponseHeadersConfig{
				HeaderNames: []string{"request-id-header"},
			},
		},
		{
			"multiple_headers",
			RequestIdResponseHeadersConfig{
				HeaderNames: []string{"request-id-header1", "request-id-header2"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cfg := DefaultRequestIdConfig
			cfg.ResponseHeaders = &tc.rh

			r := gin.New()
			r.Use(RequestId(&cfg))
			r.GET("/", func(c *gin.Context) {
				c.String(200, "ok")
			})

			w := makeRequest(t, r)
			seen := make(map[string]struct{})
			for _, name := range tc.rh.HeaderNames {
				vals := w.Header().Values(name)
				require.Len(t, vals, 1)
				seen[vals[0]] = struct{}{}
			}
			require.LessOrEqual(t, len(seen), 1)
		})
	}
}

func TestRequestId_IdGeneration(t *testing.T) {
	t.Run("uuid_v4", func(t *testing.T) {
		cfg := DefaultRequestIdConfig
		cfg.IdGeneration.Algorithm = RIDGA_UUID4

		r := gin.New()
		r.Use(RequestId(&cfg))
		r.GET("/", func(c *gin.Context) {
			reqId, ok := GetRequestId(c)
			require.True(t, ok)
			require.NotEmpty(t, reqId)
			require.NoError(t, uuid.Validate(reqId))
			c.String(200, "ok")
		})

		makeRequest(t, r)
	})
}

func TestRequestId_PassFromRequest(t *testing.T) {
	testCases := []struct {
		title string
		rhCfg RequestIdRequestHeadersConfig
		reqh  http.Header
		exp   string
	}{
		{
			"default_config",
			*DefaultRequestIdConfig.RequestHeaders,
			http.Header{
				DefaultRequestIdHeaderName: []string{"my_request_id"},
			},
			"my_request_id",
		},
		{
			"single_header__single_value",
			RequestIdRequestHeadersConfig{
				CandidateHeaderNames: []string{"request-id-header"},
				ResolutionAlgorithm:  HRA_FirstNotEmpty,
			},
			http.Header{
				"request-id-header": []string{"my_request_id"},
			},
			"my_request_id",
		},
		{
			"single_header__multiple_values",
			RequestIdRequestHeadersConfig{
				CandidateHeaderNames: []string{"request-id-header"},
				ResolutionAlgorithm:  HRA_FirstNotEmpty,
			},
			http.Header{
				"request-id-header": []string{"my_request_id1", "my_request_id2"},
			},
			"my_request_id1",
		},
		{
			"multiple_headers__second_is_found",
			RequestIdRequestHeadersConfig{
				CandidateHeaderNames: []string{"request-id-header1", "request-id-header2"},
				ResolutionAlgorithm:  HRA_FirstNotEmpty,
			},
			http.Header{
				"request-id-header2": []string{"my_request_id"},
			},
			"my_request_id",
		},
		{
			"multiple_headers__second_is_not_empty",
			RequestIdRequestHeadersConfig{
				CandidateHeaderNames: []string{"request-id-header1", "request-id-header2"},
				ResolutionAlgorithm:  HRA_FirstNotEmpty,
			},
			http.Header{
				"request-id-header1": []string{""},
				"request-id-header2": []string{"my_request_id"},
			},
			"my_request_id",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cfg := DefaultRequestIdConfig
			cfg.RequestHeaders = &tc.rhCfg

			r := gin.New()
			r.Use(RequestId(&cfg))
			r.GET("/", func(c *gin.Context) {
				reqId, ok := GetRequestId(c)
				require.True(t, ok)
				require.NotEmpty(t, reqId)
				require.Equal(t, tc.exp, reqId)
				c.String(200, "ok")
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
			req.Header = tc.reqh.Clone()
			r.ServeHTTP(w, req)
			require.Equal(t, 200, w.Code)
		})
	}
}

func TestRequestId_ProxyFromRequest(t *testing.T) {
	testCases := []struct {
		title   string
		reqCfg  RequestIdRequestHeadersConfig
		respCfg RequestIdResponseHeadersConfig
		reqh    http.Header
		expResp http.Header
	}{
		{
			"proxy_matched_request_header_to_response",
			RequestIdRequestHeadersConfig{
				CandidateHeaderNames: []string{"request-id-header1"},
				ResolutionAlgorithm:  HRA_FirstNotEmpty,
			},
			RequestIdResponseHeadersConfig{
				HeaderNames:               []string{"request-id-header2"},
				ProxyMatchedRequestHeader: ptr(true),
			},
			http.Header{
				"request-id-header1": []string{"my_request_id"},
			},
			http.Header{
				"request-id-header1": []string{"my_request_id"},
				"request-id-header2": []string{"my_request_id"},
			},
		},
		{
			"donot_proxy_matched_request_header_to_response",
			RequestIdRequestHeadersConfig{
				CandidateHeaderNames: []string{"request-id-header1"},
				ResolutionAlgorithm:  HRA_FirstNotEmpty,
			},
			RequestIdResponseHeadersConfig{
				HeaderNames:               []string{"request-id-header2"},
				ProxyMatchedRequestHeader: ptr(false),
			},
			http.Header{
				"request-id-header1": []string{"my_request_id"},
			},
			http.Header{
				"request-id-header2": []string{"my_request_id"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cfg := DefaultRequestIdConfig
			cfg.RequestHeaders = &tc.reqCfg
			cfg.ResponseHeaders = &tc.respCfg

			r := gin.New()
			r.Use(RequestId(&cfg))
			r.GET("/", func(c *gin.Context) {
				c.String(200, "ok")
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
			req.Header = tc.reqh.Clone()
			r.ServeHTTP(w, req)
			require.Equal(t, 200, w.Code)

			for name, expValues := range tc.expResp {
				require.Equal(t, expValues, w.Header().Values(name))
			}
		})
	}
}

func TestRequestId_ConfigValidation(t *testing.T) {
	testCases := []struct {
		title string
		cfg   *RequestIdMiddlewareConfig
		valid bool
	}{
		{
			"default",
			&DefaultRequestIdConfig,
			true,
		},
		{
			"minimal",
			&RequestIdMiddlewareConfig{
				IdGeneration: RequetsIdGenerationConfig{
					Algorithm: RIDGA_UUID4,
				},
			},
			true,
		},
		{
			"empty",
			&RequestIdMiddlewareConfig{},
			false,
		},
		{
			"nil",
			nil,
			false,
		},
		{
			"no_request_headers",
			&RequestIdMiddlewareConfig{
				RequestHeaders: &RequestIdRequestHeadersConfig{
					ResolutionAlgorithm: HRA_FirstNotEmpty,
				},
				IdGeneration: RequetsIdGenerationConfig{
					Algorithm: RIDGA_UUID4,
				},
			},
			false,
		},
		{
			"no_request_headers_resolution_algo",
			&RequestIdMiddlewareConfig{
				RequestHeaders: &RequestIdRequestHeadersConfig{
					CandidateHeaderNames: []string{"request-id-header"},
				},
				IdGeneration: RequetsIdGenerationConfig{
					Algorithm: RIDGA_UUID4,
				},
			},
			false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			if !tc.valid {
				require.Panics(t, func() {
					RequestId(tc.cfg)
				})
			} else {
				_ = RequestId(tc.cfg)
			}
		})
	}
}

func makeRequest(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	return w
}
