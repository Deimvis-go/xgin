package ginmw

import "github.com/Deimvis-go/xgin/ginmw/ginmwtimeout"

var Timeout = ginmwtimeout.Timeout

type TimeoutMiddlewareConfig = ginmwtimeout.MiddlewareConfig
type TimeoutRegexpRule = ginmwtimeout.RegexpRule
