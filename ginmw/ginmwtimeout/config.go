package ginmwtimeout

import (
	"github.com/Deimvis/models/utility/go/dmutil"
)

type MiddlewareConfig struct {
	DefaultTimeoutMs                int                       `yaml:"default_timeout_ms" validate:"gte=500"`
	DefaultDeadlineExpirationPolicy *DeadlineExpirationPolicy `yaml:"default_deadline_expiration_policy"`
	RegexpRules                     []RegexpRule              `yaml:"regexp_rules"`
	// TODO: option to configure order of response components (as-is / straight)
	// - as-is: proxy writes as they were from handler
	// - straight: proxy writes in the straight order of http response structure
}

type RegexpRule struct {
	PathRegexp string `yaml:"path_regexp"`
	TimeoutMs  int    `yaml:"timeout_ms"`
}

type DeadlineExpirationPolicy struct {
	// NotifyHandler cancels handler's context.
	NotifyHandler NotifyHandlerAction `yaml:"notify_handler"`
	// CloseResponse allows to immediately abort request processing
	// and send response to client (ignoring any future handler writes).
	// It MUST be used as a first middleware, because
	// it prohibits all response writes after its end.
	// Why NOT to use this:
	// 1. Unstable (no good tests, many edge cases, implementation is difficult)
	// 2. May be inefficient (double gin context allocation,
	// buffering of all response content in case
	// when response is overwriten with timeout response,
	// goroutine synchronization overhead)
	// 3. Debug difficulties (handler is launched in child goroutine,
	// which spoils stack traces and all response writer operations
	// may work not as expected, since most of them are buffered
	// and applied only in the end of request processing,
	// also there is a moment when buffered response writer swapped
	// back to the original, and during this moment all pending writes
	// would result into error that writer (buffered) is closed, but
	// consequent writes would obtain original writer and suceed)
	CloseResponse CloseResponseAction `yaml:"close_response"`
}

type NotifyHandlerAction struct {
	dmutil.Option `yaml:",inline"`
}

type CloseResponseAction struct {
	dmutil.Option              `yaml:",inline"`
	OverwriteToTimeoutResponse bool `yaml:"overwrite_to_timeout_response"`
}
