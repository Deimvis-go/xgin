# xgin

xgin - golang library with extensions for the [Gin](https://github.com/gin-gonic/gin) web framework:

## Install

```bash
go get github.com/Deimvis-go/xgin
```

## Middlewares

### Request Id

```go
r.Use(ginmw.RequestId(&ginmw.DefaultRequestIdConfig))

r.GET("/ping", func(c *gin.Context) {
    reqId, _ := ginmw.GetRequestId(c)
    c.JSON(200, gin.H{"request_id": reqId})
})
```

### Timeout

```go
r.Use(ginmw.Timeout(&ginmw.TimeoutMiddlewareConfig{
    DefaultTimeoutMs: 5000,
    RegexpRules: []ginmw.TimeoutRegexpRule{
        {PathRegexp: "/upload/.*", TimeoutMs: 60000},
        {PathRegexp: "/health",    TimeoutMs: 500},
    },
}, logger))
```

### Recovery

```go
r.Use(ginmw.Recovery(logger, map[any]ginmw.ErrorHandlerFunc{
    ErrNotFound{}:   func(c *gin.Context, err error) { c.AbortWithStatusJSON(404, gin.H{"error": err.Error()}) },
    ErrBadRequest{}: func(c *gin.Context, err error) { c.AbortWithStatusJSON(400, gin.H{"error": err.Error()}) },
}))
```

### Prometheus

```go
r.Use(ginmw.Prometheus(ginmw.PrometheusConfig{
    StartC:    startCounter,
    FinishC:   finishCounter,
    DurationH: durationHistogram,
    ExtraLabels: []ginmw.PrometheusExtraLabel{
        {Name: "client_ip", Value: func(c *gin.Context) string { return c.ClientIP() }},
    },
}))
```

## Handler

```go
func Create(ctx context.Context, req *CreateRequest) fw.Response {
    item := service.Create(ctx, req)
    return fw.JSON(201, item)
}

// automatically decodes uri, query and body
r.POST("/v1/items", ginss.NewHandler(items.Create)))
```

## Context Decoding

```go
func myMiddleware(c *gin.Context) {
    // add callback to decode string key of gin.Context to any key of context.Context.
    ginctx.AddDecodeCallback(c, ginctx.ShouldDecodeKey("user_id", userIdKey{}))
    c.Next()
}

r.GET("/profile", func(c *gin.Context) {
    ctx, _ := ginctx.Decode(c)
    userId := ctx.Value(userIdKey{}).(string)
    ...
})
```

