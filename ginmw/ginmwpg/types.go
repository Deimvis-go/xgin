// Package ginmwpg provides Gin middleware that acquires a Postgres
// connection per request and stores it in the gin context. Handlers retrieve
// the connection via [CtxConn] using the same [Mode].
//
// The package does not depend on a concrete pg driver. Any connection layer
// that satisfies [Provider] can be plugged in.
package ginmwpg

import (
	"context"

	"github.com/Deimvis/go-ext/go1.25/xoptional"
)

// Mode selects between read-only and read-write connections.
type Mode string

const (
	RO Mode = "ro"
	RW Mode = "rw"
)

// String implements [fmt.Stringer].
func (m Mode) String() string { return string(m) }

// AcquireOption is an opaque option forwarded to the [Provider]. The
// middleware does not inspect options — callers should pass whatever option
// type their provider expects.
type AcquireOption = any

// Provider acquires a connection in the requested [Mode]. The returned
// connection is opaque to the middleware — handlers cast it to the concrete
// type their pg layer exposes. The optional [Ownership] is returned when the
// caller is responsible for freeing the connection; if absent, the
// connection is managed externally.
type Provider interface {
	Acquire(ctx context.Context, mode Mode, opts ...AcquireOption) (any, xoptional.T[Ownership], error)
}

// Ownership wraps the right to release an acquired connection. The
// middleware calls [Ownership.MustTake] exactly once and releases the
// resulting [OwnedConn] when the request completes.
type Ownership interface {
	MustTake() OwnedConn
}

// OwnedConn is a connection whose ownership has been taken; [FreeConn] must
// be called to release it back to the underlying pool.
type OwnedConn interface {
	FreeConn(context.Context) error
}
