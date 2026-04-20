package ginmw

import "errors"

// RequestIdMiddlewareConfig configures the [RequestId] middleware.
type RequestIdMiddlewareConfig struct {
	RequestHeaders  *RequestIdRequestHeadersConfig  `yaml:"request_headers"`
	ResponseHeaders *RequestIdResponseHeadersConfig `yaml:"response_headers"`
	Context         *RequestIdGoContextConfig       `yaml:"go_context"`
	IdGeneration    RequetsIdGenerationConfig       `yaml:"id_generation"`
}

// RequestIdRequestHeadersConfig describes how the middleware extracts an
// incoming request id from request headers.
type RequestIdRequestHeadersConfig struct {
	CandidateHeaderNames []string                   `yaml:"candidate_header_names"`
	ResolutionAlgorithm  HeadersResolutionAlgorithm `yaml:"resolution_algorithm"`
}

// RequestIdResponseHeadersConfig describes how the middleware exposes the
// request id to the response.
type RequestIdResponseHeadersConfig struct {
	HeaderNames []string `yaml:"header_names"`
	// ProxyMatchedRequestHeader, when set to a true pointer, copies the
	// matched request header name/value pair into the response.
	ProxyMatchedRequestHeader *bool `yaml:"proxy_matched_request_header"`
}

// RequestIdGoContextConfig exposes the request id under additional gin
// context keys.
type RequestIdGoContextConfig struct {
	ExtraKeys []string `yaml:"extra_keys"`
}

// RequetsIdGenerationConfig configures request id generation when no header
// match is found.
type RequetsIdGenerationConfig struct {
	Algorithm RequestIdGenerationAlgorithm `yaml:"algorithm"`
}

// HeadersResolutionAlgorithm names an algorithm to pick a header value from
// the incoming request.
type HeadersResolutionAlgorithm string

const (
	// HRA_FirstNotEmpty picks the first candidate header with a non-empty value.
	HRA_FirstNotEmpty HeadersResolutionAlgorithm = "first_not_empty"
)

// RequestIdGenerationAlgorithm names an algorithm to generate a new
// request id.
type RequestIdGenerationAlgorithm string

const (
	// RIDGA_UUID4 generates a random UUIDv4.
	RIDGA_UUID4 RequestIdGenerationAlgorithm = "uuid_v4"
)

// validate returns an error if the config is not usable.
func (cfg *RequestIdMiddlewareConfig) validate() error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.IdGeneration.Algorithm == "" {
		return errors.New("id_generation.algorithm is required")
	}
	if cfg.RequestHeaders != nil {
		if len(cfg.RequestHeaders.CandidateHeaderNames) == 0 {
			return errors.New("request_headers.candidate_header_names must not be empty")
		}
		if cfg.RequestHeaders.ResolutionAlgorithm == "" {
			return errors.New("request_headers.resolution_algorithm is required")
		}
	}
	return nil
}
