package ginmw

type RequestIdMiddlewareConfig struct {
	RequestHeaders  *RequestIdRequestHeadersConfig  `yaml:"request_headers" validate:"omitnil"`
	ResponseHeaders *RequestIdResponseHeadersConfig `yaml:"response_headers" validate:"omitnil"`
	Context         *RequestIdGoContextConfig       `yaml:"go_context" validate:"omitnil"`
	IdGeneration    RequetsIdGenerationConfig       `yaml:"id_generation"`
}

type RequestIdRequestHeadersConfig struct {
	CandidateHeaderNames []string                   `yaml:"candidate_header_names" validate:"gt=0"`
	ResolutionAlgorithm  HeadersResolutionAlgorithm `yaml:"resolution_algorithm" validate:"required"`
}

type RequestIdResponseHeadersConfig struct {
	HeaderNames               []string `yaml:"header_names"`
	ProxyMatchedRequestHeader *bool    `yaml:"proxy_matched_request_header"`
}

type RequestIdGoContextConfig struct {
	ExtraKeys []string `yaml:"extra_keys"`
}

type RequetsIdGenerationConfig struct {
	Algorithm RequestIdGenerationAlgorithm `yaml:"algorithm" validate:"required"`
}

type HeadersResolutionAlgorithm string

var (
	HRA_FirstNotEmpty HeadersResolutionAlgorithm = "first_not_empty"
)

type RequestIdGenerationAlgorithm string

var (
	RIDGA_UUID4 RequestIdGenerationAlgorithm = "uuid_v4"
)
