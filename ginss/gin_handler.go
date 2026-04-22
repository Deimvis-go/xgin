package ginss

import (
	"context"
	"io"
	"reflect"

	"github.com/Deimvis-go/fw/fw"
	"github.com/Deimvis-go/logs/logs"
	"github.com/Deimvis-go/valid"
	"github.com/Deimvis/go-ext/go1.25/ext"
	"github.com/Deimvis/go-ext/go1.25/xcheck"
	"github.com/Deimvis/go-ext/go1.25/xcheck/xmust"
	"github.com/gin-gonic/gin"
)

// TODO: make logger an option (hence run win no logger by default)
// It will also force users to create []GinHandlerOption in order to pass them into underlying functions:
// hopts := []ginss.GinHandlerOption{ginss.WithLogger(logger), ginss.WithContextDecoding(decodeGinContext)}
// InitAuthHandlers(router, hopts)
// > hopts = append(hopts, ginss.WithNoLogging())
// > router.GET("/v1/auth/login", ginss.NewHandler(auth.Login, hopts...))

// NewHandler is a shortcut for request and response encoding.
// NOTE: fw.Request implementation should implement an interface by pointer.
func NewHandler[Request fw.RequestReflect](
	fn func(ctx context.Context, req Request) fw.Response,
	opts ...GinHandlerOption,
) gin.HandlerFunc {
	cfg := defaultGinHandlerConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	reqType := reflect.TypeOf(new(Request)).Elem()
	xmust.Eq(reqType.Kind(), reflect.Pointer, "fw.Request implementation must be a pointer", xcheck.PrintWhy())
	return func(c *gin.Context) {
		req := reflect.New(reqType.Elem()).Interface().(Request)

		ctx := xmust.Do(cfg.DecodeGinContext(c))
		ctx = ConcealGinContext(ctx, c)
		lgFns := cfg.bindContextAndLogger(ctx, cfg.Logger)

		uriPtr := xmust.Do(ext.GetFieldAddr(req, "URI"))
		if uriPtr != nil {
			if !BindOr400(c, c.ShouldBindUri, uriPtr, lgFns.LogRequestParseError) {
				return // 400
			}
		}
		queryPtr := xmust.Do(ext.GetFieldAddr(req, "Query"))
		if queryPtr != nil {
			if !BindOr400(c, c.ShouldBindQuery, queryPtr, lgFns.LogRequestParseError) {
				return // 400
			}
		}
		if !SetOr400(c, req.SetHeader, c.Request.Header, lgFns.LogRequestParseError) {
			return // 400
		}
		if !SetOr400(c, req.SetBodyStream, io.Reader(c.Request.Body), lgFns.LogRequestParseError) {
			return // 400
		}

		lgFns.LogRequest(req)
		if !IsValidRespOr400(c, req, ctx, cfg.Logger) {
			return // 400
		}
		resp := fn(ctx, req)
		// legacy: filling nil arrays (TODO: remove)
		v := reflect.ValueOf(resp)
		if v.Kind() == reflect.Pointer {
			ext.FillNilSlices(resp)
		}
		if !IsValidRespOr500(c, resp, ctx, cfg.Logger) {
			return // 500
		}
		lgFns.LogResponse(req, resp)

		c.Status(resp.Code())
		resp.Header().Range(func(k string, vs []string) bool {
			for _, v := range vs {
				c.Writer.Header().Add(k, v)
			}
			return true
		})
		switch r := resp.(type) {
		case fw.BufferedResponse:
			c.Writer.Write(r.BodyRaw())
		default:
			_, err := io.Copy(c.Writer, r.BodyStream())
			if err != nil {
				cfg.Logger.Info(ctx, "Failed to write response stream", "err", err)
				c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
				return // 500
			}
		}
	}
}

func IsValidRespOr400(c *gin.Context, obj interface{}, logctx context.Context, lg logs.KVCtxLogger) bool {
	err := valid.Deep(obj)
	if err != nil {
		lg.Info(logctx, "Request is invalid", "err", err)
		c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func IsValidRespOr500(c *gin.Context, obj interface{}, logctx context.Context, lg logs.KVCtxLogger) bool {
	err := valid.Deep(obj)
	if err != nil {
		lg.Info(logctx, "Response is invalid", "err", err)
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return false
	}
	return true
}
