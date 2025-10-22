package pbjs

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	ctxKeyHeader   struct{}
	ctxKeyMetadata struct{}
)

// SetContextHeader returns a context with the [nats.Header] set
func SetContextHeader(ctx context.Context, h nats.Header) context.Context {
	return context.WithValue(ctx, ctxKeyHeader{}, h)
}

// GetContextHeader returns the [nats.Header] from the supplied context.
// If the context does not have a header set, an empty one is returned.
func GetContextHeader(ctx context.Context) nats.Header {
	if h, ok := ctx.Value(ctxKeyHeader{}).(nats.Header); ok {
		return h
	}
	return make(nats.Header)
}

// SetContextMetadata returns a context with the [jetstream.MsgMetadata] set
func SetContextMetadata(ctx context.Context, md *jetstream.MsgMetadata) context.Context {
	return context.WithValue(ctx, ctxKeyMetadata{}, md)
}

// GetContextMetadata returns the [jetstream.MsgMetadata] from the supplied context.
// If the context does not have metadata set, an empty one is returned.
func GetContextMetadata(ctx context.Context) *jetstream.MsgMetadata {
	if m, ok := ctx.Value(ctxKeyMetadata{}).(*jetstream.MsgMetadata); ok {
		return m
	}
	return &jetstream.MsgMetadata{}
}
