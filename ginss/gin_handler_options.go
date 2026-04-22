package ginss

import (
	"context"

	"github.com/Deimvis-go/fw/fw"
	"github.com/Deimvis-go/logs/logs"
	"github.com/Deimvis-go/xgin/ginctx"
	"github.com/gin-gonic/gin"
)

type GinHandlerConfig struct {
	Logger               logs.KVCtxLogger
	LogRequestParseError func(context.Context, logs.KVCtxLogger, error)
	LogRequest           func(context.Context, logs.KVCtxLogger, fw.Request)
	LogResponse          func(context.Context, logs.KVCtxLogger, fw.Request, fw.Response)
	// TODO: LogInvalidRequest, LogInvalidResponse
	DecodeGinContext func(*gin.Context) (context.Context, error)
}

var defaultGinHandlerConfig = GinHandlerConfig{
	Logger: logs.NoopKVLogger.KVCtx(),
	LogRequestParseError: func(ctx context.Context, lg logs.KVCtxLogger, err error) {
		lg.Info(ctx, "Bad request", "err", err)
	},
	LogRequest: func(ctx context.Context, lg logs.KVCtxLogger, req fw.Request) {
		switch r := req.(type) {
		case fw.BufferedRequest:
			lg.Debug(ctx, "Request", "method", r.Method(), "path", r.Path(), "query", r.QueryString(), "body", string(r.BodyRaw()))
		default:
			lg.Debug(ctx, "Request", "method", r.Method(), "path", r.Path(), "query", r.QueryString())
		}
	},
	LogResponse: func(ctx context.Context, lg logs.KVCtxLogger, req fw.Request, resp fw.Response) {
		switch r := resp.(type) {
		case fw.BufferedResponse:
			lg.Debug(ctx, "Response", "code", r.Code(), "body", string(r.BodyRaw()), "req_method", req.Method(), "req_path", req.Path())
		default:
			lg.Debug(ctx, "Response", "code", r.Code(), "req_method", req.Method(), "req_path", req.Path())
		}
	},
	DecodeGinContext: ginctx.Decode,
}

type GinHandlerOption func(cfg *GinHandlerConfig)

func WithLogger(lg logs.KVCtxLogger) GinHandlerOption {
	return func(cfg *GinHandlerConfig) {
		cfg.Logger = lg
	}
}

// also does not log headers
// TODO: make more transparent off-the-shelf logging options to manage what to log (headers, body, etc)
func WithNoBodyLogging() GinHandlerOption {
	return func(cfg *GinHandlerConfig) {
		cfg.LogRequest = func(ctx context.Context, lg logs.KVCtxLogger, req fw.Request) {
			lg.Debug(ctx, "Request", "method", req.Method(), "path", req.Path(), "query", req.QueryString())
		}
		cfg.LogResponse = func(ctx context.Context, lg logs.KVCtxLogger, req fw.Request, resp fw.Response) {
			lg.Debug(ctx, "Response", "code", resp.Code(), "req_method", req.Method(), "req_path", req.Path())
		}
	}
}

func WithNoLogging() GinHandlerOption {
	return func(cfg *GinHandlerConfig) {
		cfg.LogRequest = func(context.Context, logs.KVCtxLogger, fw.Request) {}
		cfg.LogResponse = func(context.Context, logs.KVCtxLogger, fw.Request, fw.Response) {}
	}
}

func WithContextDecoding(fn func(*gin.Context) (context.Context, error)) GinHandlerOption {
	return func(cfg *GinHandlerConfig) {
		cfg.DecodeGinContext = fn
	}
}

func (cfg *GinHandlerConfig) bindContextAndLogger(ctx context.Context, lg logs.KVCtxLogger) *ginHandlerConfigLogFns {
	return &ginHandlerConfigLogFns{
		LogRequestParseError: func(err error) {
			cfg.LogRequestParseError(ctx, lg, err)
		},
		LogRequest: func(req fw.Request) {
			cfg.LogRequest(ctx, lg, req)
		},
		LogResponse: func(req fw.Request, resp fw.Response) {
			cfg.LogResponse(ctx, lg, req, resp)
		},
	}
}

type ginHandlerConfigLogFns struct {
	LogRequestParseError func(error)
	LogRequest           func(fw.Request)
	LogResponse          func(fw.Request, fw.Response)
}
