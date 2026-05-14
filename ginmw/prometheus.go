package ginmw

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/Deimvis-go/ms/ms/msconfig"
	"github.com/Deimvis-go/xprometheus/prom"
	"github.com/Deimvis/go-ext/go1.25/xfallback/xstringfb"
)

// TODO: add label "is_no_route" in order detect 404 requests to not registered handlers
//       it will also require to add custom gin.NoRoute handler that will pass corresponding flag to context.

// Prometheus metrics must have the following labels:
// startC:    ["method", "path", "client_host", "client_cloud_service", "client_cloud_instance", "client_ip"]
// finishC:   ["method", "path", "client_host", "client_cloud_service", "client_cloud_instance", "client_ip", "code"]
// durationH: ["method", "path", "client_host", "client_cloud_service", "client_cloud_instance", "client_ip", "code"]
func Prometheus(
	startC *prometheus.CounterVec,
	finishC *prometheus.CounterVec,
	durationH prom.DurationHistogramVec,
) func(c *gin.Context) {
	g := prom.NewIntervalGroup(startC, finishC, durationH)
	fb := func(s string) string {
		return xstringfb.OnEmpty(s, prom.LabelUnknown)
	}
	return func(c *gin.Context) {
		ls := prometheus.Labels{
			"method": c.Request.Method,
			"path":   fb(c.FullPath()),

			"client_host":           fb(c.Request.Header.Get(msconfig.XClientHost)),
			"client_cloud_service":  fb(c.Request.Header.Get(msconfig.XClientCloudService)),
			"client_cloud_instance": fb(c.Request.Header.Get(msconfig.XClientCloudInstance)),
			"client_ip":             fb(c.ClientIP()),
		}
		g.Record(ls, func(rc prom.RecordControl) {
			c.Next()
			rc.AddLabels(prometheus.Labels{
				"code": strconv.Itoa(c.Writer.Status()),
			})
		})
	}
}
