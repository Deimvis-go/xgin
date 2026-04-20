package ginmw

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusLabelUnknown is the fallback value written into string labels
// when the underlying source is empty.
const PrometheusLabelUnknown = "unknown"

// PrometheusExtraLabel extends the default label set with a custom label
// extracted from the request.
type PrometheusExtraLabel struct {
	// Name is the label name. It must match the label name declared on the
	// underlying collectors.
	Name string
	// Value returns the label value for the given request. An empty string
	// is replaced with [PrometheusLabelUnknown].
	Value func(c *gin.Context) string
}

// PrometheusConfig configures the [Prometheus] middleware.
//
// Collectors must be registered with the following base labels:
//   - StartC:    ["method", "path"] + ExtraLabels
//   - FinishC:   ["method", "path", "code"] + ExtraLabels
//   - DurationH: ["method", "path", "code"] + ExtraLabels
type PrometheusConfig struct {
	// StartC is incremented when a request enters the middleware.
	StartC *prometheus.CounterVec
	// FinishC is incremented when a request leaves the middleware.
	FinishC *prometheus.CounterVec
	// DurationH records the duration of each request, in seconds.
	DurationH *prometheus.HistogramVec
	// ExtraLabels extends the default label set. Every label declared here
	// must also be declared on the collectors above.
	ExtraLabels []PrometheusExtraLabel
}

// Prometheus returns a middleware that records per-request Prometheus metrics.
func Prometheus(cfg PrometheusConfig) gin.HandlerFunc {
	fb := func(s string) string {
		if s == "" {
			return PrometheusLabelUnknown
		}
		return s
	}
	return func(c *gin.Context) {
		labels := prometheus.Labels{
			"method": c.Request.Method,
			"path":   fb(c.FullPath()),
		}
		for _, l := range cfg.ExtraLabels {
			labels[l.Name] = fb(l.Value(c))
		}

		start := time.Now()
		cfg.StartC.With(labels).Inc()

		c.Next()

		labels["code"] = strconv.Itoa(c.Writer.Status())
		cfg.FinishC.With(labels).Inc()
		cfg.DurationH.With(labels).Observe(time.Since(start).Seconds())
	}
}
