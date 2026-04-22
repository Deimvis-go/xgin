package ginss

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetOr400[T any](c *gin.Context, setFn func(T) error, obj T, lgFn func(error)) bool {
	err := setFn(obj)
	if err != nil {
		lgFn(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bad Request: %s", err)})
		return false
	}
	return true
}

func BindOr400(c *gin.Context, bindFn func(any) error, dst any, lgFn func(error)) bool {
	err := bindFn(dst)
	if err != nil {
		lgFn(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bad Request: %s", err)})
		return false
	}
	return true
}
