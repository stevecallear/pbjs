package pbjs

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
)

type (
	// Handler represents an untyped message handler
	Handler interface {
		Handle(ctx context.Context, m proto.Message) error
	}

	// HandlerFunc represents an untyped message handler func
	HandlerFunc func(ctx context.Context, m proto.Message) error
)

// NewHandler returns a new typed message handler for the specified func
func NewHandler[T proto.Message](fn func(ctx context.Context, m T) error) Handler {
	return HandlerFunc(func(ctx context.Context, m proto.Message) error {
		tm, ok := m.(T)
		if !ok {
			return NewPersistentError(fmt.Errorf("invalid message type for handler: %T", m))
		}
		return fn(ctx, tm)
	})
}

// Handle handles the message
func (h HandlerFunc) Handle(ctx context.Context, m proto.Message) error {
	return h(ctx, m)
}

// MessageType returns the type for the supplied message
func MessageType(m proto.Message) string {
	return string(m.ProtoReflect().Descriptor().FullName())
}
