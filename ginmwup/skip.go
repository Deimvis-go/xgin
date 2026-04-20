// Package ginmwup provides generic wrappers ("upgrades") for Gin
// middlewares — for example, skipping a middleware for certain paths or
// adding debug logging around it.
package ginmwup

import (
	"regexp"

	"github.com/gin-gonic/gin"
)

// WithSkipPathRe wraps a middleware so that it is skipped for request paths
// matching any of the provided regular expressions.
func WithSkipPathRe(pathRegexps []*regexp.Regexp) func(gin.HandlerFunc) gin.HandlerFunc {
	return func(mw gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			path := c.Request.URL.Path
			for _, re := range pathRegexps {
				if re.MatchString(path) {
					c.Next()
					return
				}
			}
			mw(c)
		}
	}
}
