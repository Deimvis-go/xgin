// Package ginmwtimeout implements a request timeout middleware for Gin.
package ginmwtimeout

// MiddlewareConfig configures the [Timeout] middleware.
type MiddlewareConfig struct {
	// DefaultTimeoutMs is the timeout applied when no [RegexpRule] matches.
	DefaultTimeoutMs int `yaml:"default_timeout_ms"`
	// DefaultDeadlineExpirationPolicy controls what happens when the
	// deadline expires. When nil, [DefaultDeadlineExpirationPolicy] is used.
	DefaultDeadlineExpirationPolicy *DeadlineExpirationPolicy `yaml:"default_deadline_expiration_policy"`
	// RegexpRules lets callers override the timeout for specific paths.
	// The first matching rule wins.
	RegexpRules []RegexpRule `yaml:"regexp_rules"`
}

// RegexpRule overrides [MiddlewareConfig.DefaultTimeoutMs] for request paths
// matching PathRegexp.
type RegexpRule struct {
	PathRegexp string `yaml:"path_regexp"`
	TimeoutMs  int    `yaml:"timeout_ms"`
}

// DeadlineExpirationPolicy describes how the middleware reacts when the
// request deadline expires.
type DeadlineExpirationPolicy struct {
	// NotifyHandler cancels the request's context on deadline expiration.
	NotifyHandler NotifyHandlerAction `yaml:"notify_handler"`
	// CloseResponse is reserved for a future implementation that aborts
	// the request and sends a timeout response while forbidding further
	// writes from the handler. It is currently not supported on Gin — see
	// the package doc for details.
	CloseResponse CloseResponseAction `yaml:"close_response"`
}

// NotifyHandlerAction toggles the cancel-context behavior.
type NotifyHandlerAction struct {
	Enabled *bool `yaml:"enabled"`
}

// IsEnabled reports whether the action is enabled.
func (a NotifyHandlerAction) IsEnabled() bool { return a.Enabled != nil && *a.Enabled }

// CloseResponseAction toggles the close-response behavior.
type CloseResponseAction struct {
	Enabled                    *bool `yaml:"enabled"`
	OverwriteToTimeoutResponse bool  `yaml:"overwrite_to_timeout_response"`
}

// IsEnabled reports whether the action is enabled.
func (a CloseResponseAction) IsEnabled() bool { return a.Enabled != nil && *a.Enabled }
