package ginmwup

import (
	"regexp"

	"github.com/gin-gonic/gin"
)

func WithSkipPathRe(pathRegexps []*regexp.Regexp) func(gin.HandlerFunc) gin.HandlerFunc {
	return func(mw gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			path := c.Request.URL.Path
			for _, re := range pathRegexps {
				if re.Match([]byte(path)) {
					c.Next()
					return
				}
			}
			mw(c)
		}
	}
}
