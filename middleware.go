package pbjs

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/proto"
)

// MiddlewareFunc represents a publisher/consumer middleware func
type MiddlewareFunc func(h Handler) Handler

// ApplyMiddleware returns a handler with the supplied middleware applied
func ApplyMiddleware(h Handler, m ...MiddlewareFunc) Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

// LoggingMiddleware returns a [slog.Logger] middleware funct
func LoggingMiddleware(l *slog.Logger) MiddlewareFunc {
	return func(h Handler) Handler {
		return HandlerFunc(func(ctx context.Context, m proto.Message) error {
			hd := GetContextHeader(ctx)

			hl := l.With(slog.String(HdrNatsMsgID, hd.Get(HdrNatsMsgID)), slog.String(HdrMsgType, hd.Get(HdrMsgType)))
			hl.LogAttrs(ctx, slog.LevelDebug, "message received")

			if err := h.Handle(ctx, m); err != nil {
				hl.LogAttrs(ctx, slog.LevelError, "message failure", slog.String("err", err.Error()))
				return err
			}

			hl.LogAttrs(ctx, slog.LevelInfo, "message success")
			return nil
		})
	}
}
