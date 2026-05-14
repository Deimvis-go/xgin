package ginmwpg

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	pgx "github.com/jackc/pgx/v5"
	pgxconn "github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"github.com/Deimvis-go/xgin/ginmw/ginmwtimeout"
	"github.com/Deimvis-go/xgin/ginmw/internal/mwtest"
	"github.com/Deimvis-go/xpg/pg"
	"github.com/Deimvis-go/xpg/pg/pgconn"
	"github.com/Deimvis/go-ext/go1.25/xoptional"
	"github.com/Deimvis/go-ext/go1.25/xptr"
	"github.com/Deimvis/models/utility/go/dmutil"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type IConn interface {
	Exec(ctx context.Context, sql string, args ...any) (commandTag pgxconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type OwnedConn interface {
	FreeConn(context.Context) error
}

type ConnOwnership interface {
	Take() (pgconn.OwnedConn, error)
	MustTake() pgconn.OwnedConn
}

type ConnProvider interface {
	Acquire(context.Context, pgconn.Mode, ...pgconn.AcquireOption) (pg.Conn, xoptional.T[pg.ConnOwnership], error)
	AcquireManaged(context.Context, pgconn.Mode, ...pgconn.AcquireOption) (pg.Conn, error)
}

//go:generate mockgen -source=conn_test.go -destination=conn_mocks_test.go -package=ginmwpg

func TestMain(m *testing.M) {
	ginWriter := gin.DefaultWriter
	gin.DefaultWriter = io.Discard

	code := m.Run()

	gin.DefaultWriter = ginWriter

	os.Exit(code)
}

func TestConn(t *testing.T) {
	t.Run("conn-available", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := NewMockIConn(ctrl)
		cp := NewMockConnProvider(ctrl)
		cp.EXPECT().
			Acquire(gomock.Any(), pgconn.RO, gomock.Any()).
			Return(conn, xoptional.New[pg.ConnOwnership](), nil).
			Times(1)

		mw := Conn(pgconn.RO, cp)
		h := func(c *gin.Context) {
			conn := pg.CtxConn(c, pgconn.RO)
			require.NotNil(t, conn)
		}
		invokeHandler(h, mw)
	})
	t.Run("conn-released", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := NewMockIConn(ctrl)
		ownedConn := NewMockOwnedConn(ctrl)
		own := NewMockConnOwnership(ctrl)
		cp := NewMockConnProvider(ctrl)
		cp.EXPECT().
			Acquire(gomock.Any(), pgconn.RO, gomock.Any()).
			Return(conn, xoptional.New[pg.ConnOwnership](own), nil).
			Times(1)
		own.EXPECT().MustTake().Return(ownedConn).Times(1)
		release := ownedConn.EXPECT().FreeConn(gomock.Any())
		release.Return(nil).Times(1)

		mw := Conn(pgconn.RO, cp)
		h := func(c *gin.Context) {}
		invokeHandler(h, mw)
	})
	t.Run("panic/conn-released", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := NewMockIConn(ctrl)
		ownedConn := NewMockOwnedConn(ctrl)
		own := NewMockConnOwnership(ctrl)
		cp := NewMockConnProvider(ctrl)
		cp.EXPECT().
			Acquire(gomock.Any(), pgconn.RO, gomock.Any()).
			Return(conn, xoptional.New[pg.ConnOwnership](own), nil).
			Times(1)
		own.EXPECT().MustTake().Return(ownedConn).Times(1)
		release := ownedConn.EXPECT().FreeConn(gomock.Any())
		release.Return(nil).Times(1)

		mw := Conn(pgconn.RO, cp)
		h := func(c *gin.Context) {
			panic(1)
		}
		require.Panics(t, func() {
			invokeHandler(h, mw)
		})
	})
}

func TestConn_Integration(t *testing.T) {
	timeoutMwCfg := ginmwtimeout.MiddlewareConfig{
		DefaultTimeoutMs: 1000,
		DefaultDeadlineExpirationPolicy: &ginmwtimeout.DeadlineExpirationPolicy{
			NotifyHandler: ginmwtimeout.NotifyHandlerAction{
				Option: dmutil.Option{Enabled: xptr.T(true)},
			},
		},
	}
	newTimeoutMw := func(t *testing.T) (gin.HandlerFunc, chan<- int, func()) {
		advanceTimeout := make(chan int, 1)
		mwtest.TimeAfter = func(d time.Duration) <-chan time.Time {
			ch := make(chan time.Time, 1)
			go func() {
				<-advanceTimeout
				ch <- time.Now()
				close(ch)
			}()
			return ch
		}
		clean := func() {
			close(advanceTimeout)
			mwtest.TimeAfter = time.After
		}
		mw := ginmwtimeout.Timeout(&timeoutMwCfg, zap.S())
		return mw, advanceTimeout, clean
	}
	t.Run("timeoutmw_notify-handler/deadline-met/conn-released", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := NewMockIConn(ctrl)
		ownedConn := NewMockOwnedConn(ctrl)
		own := NewMockConnOwnership(ctrl)
		cp := NewMockConnProvider(ctrl)
		cp.EXPECT().
			Acquire(gomock.Any(), pgconn.RO, gomock.Any()).
			Return(conn, xoptional.New[pg.ConnOwnership](own), nil).
			Times(1)
		own.EXPECT().MustTake().Return(ownedConn).Times(1)
		release := ownedConn.EXPECT().FreeConn(gomock.Any())
		release.Return(nil).Times(1)

		timeoutMw, _, clean := newTimeoutMw(t)
		defer clean()
		mw := Conn(pgconn.RO, cp)
		h := func(c *gin.Context) {}
		invokeHandler(h, timeoutMw, mw)
	})
	t.Run("timeoutmw_notify-handler/deadline-met/panic/conn-released", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := NewMockIConn(ctrl)
		ownedConn := NewMockOwnedConn(ctrl)
		own := NewMockConnOwnership(ctrl)
		cp := NewMockConnProvider(ctrl)
		cp.EXPECT().
			Acquire(gomock.Any(), pgconn.RO, gomock.Any()).
			Return(conn, xoptional.New[pg.ConnOwnership](own), nil).
			Times(1)
		own.EXPECT().MustTake().Return(ownedConn).Times(1)
		release := ownedConn.EXPECT().FreeConn(gomock.Any())
		release.Return(nil).Times(1)

		timeoutMw, _, clean := newTimeoutMw(t)
		defer clean()
		mw := Conn(pgconn.RO, cp)
		h := func(c *gin.Context) {
			panic(1)
		}
		require.Panics(t, func() {
			invokeHandler(h, timeoutMw, mw)
		})
	})
	t.Run("timeoutmw_notify-handler/deadline-expired/conn-released", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := NewMockIConn(ctrl)
		ownedConn := NewMockOwnedConn(ctrl)
		own := NewMockConnOwnership(ctrl)
		cp := NewMockConnProvider(ctrl)
		cp.EXPECT().
			Acquire(gomock.Any(), pgconn.RO, gomock.Any()).
			Return(conn, xoptional.New[pg.ConnOwnership](own), nil).
			Times(1)
		own.EXPECT().MustTake().Return(ownedConn).Times(1)
		release := ownedConn.EXPECT().FreeConn(gomock.Any())
		release.Return(nil).Times(1)

		advanceHandler := make(chan int, 1)
		defer close(advanceHandler)
		timeoutMw, _, clean := newTimeoutMw(t)
		defer clean()
		expireMw := func(c *gin.Context) {
			newCtx, _ := context.WithTimeout(c.Request.Context(), -time.Nanosecond)
			c.Request = c.Request.WithContext(newCtx)
		}
		mw := Conn(pgconn.RO, cp)
		h := func(c *gin.Context) {
			<-c.Request.Context().Done()
		}
		invokeHandler(h, timeoutMw, expireMw, mw)
	})
}

func invokeHandler(h func(c *gin.Context), mws ...gin.HandlerFunc) *httptest.ResponseRecorder {
	r := gin.New()
	r.ContextWithFallback = true
	r.Use(mws...)
	r.GET("/", h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	r.ServeHTTP(w, req)
	return w
}
