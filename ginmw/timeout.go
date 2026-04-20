package ginmw

import "github.com/Deimvis-go/xgin/ginmw/ginmwtimeout"

// Timeout is an alias for [ginmwtimeout.Timeout].
var Timeout = ginmwtimeout.Timeout

// TimeoutMiddlewareConfig is an alias for [ginmwtimeout.MiddlewareConfig].
type TimeoutMiddlewareConfig = ginmwtimeout.MiddlewareConfig

// TimeoutRegexpRule is an alias for [ginmwtimeout.RegexpRule].
type TimeoutRegexpRule = ginmwtimeout.RegexpRule
