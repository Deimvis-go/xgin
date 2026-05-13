package ginmwpg

import "context"

// CtxConn returns the connection stored in ctx for the given [Mode], or nil
// if no connection has been acquired for that mode.
//
// ctx may be a [*gin.Context] or any context derived from one (e.g. via the
// ginctx package).
func CtxConn(ctx context.Context, mode Mode) any {
	v := ctx.Value(ctxConnKey(mode))
	if v == nil {
		return nil
	}
	return v
}

// CtxConnKey returns the context key used to store a connection acquired in
// the given [Mode]. Exposed so that callers can mock or inject connections
// in tests.
func CtxConnKey(mode Mode) any { return ctxConnKey(mode) }

func ctxConnKey(mode Mode) string {
	switch mode {
	case RO:
		return ctxConnKeyRO
	case RW:
		return ctxConnKeyRW
	}
	return "ginmwpg.conn_" + string(mode)
}

const (
	ctxConnKeyRO = "ginmwpg.conn_ro__zZ5N5pXn8h7uCXjxTpL"
	ctxConnKeyRW = "ginmwpg.conn_rw__zZ5N5pXn8h7uCXjxTpL"
)
