package pbjs

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
)

type (
	// MsgPtr represents a pointer to a proto.Message implementation
	MsgPtr[T any] interface {
		*T
		proto.Message
	}

	// Handler represents an untyped message handler
	Handler interface {
		Handle(ctx context.Context, m proto.Message) error
	}

	// HandlerFunc represents an untyped message handler func
	HandlerFunc func(ctx context.Context, m proto.Message) error

	// TypeHandler represents a typed message handler
	TypeHandler interface {
		Handler
		Type() string
	}

	typeHandler[T proto.Message] struct {
		mtype string
		fn    func(ctx context.Context, m T) error
	}
)

// NewHandler returns a new typed message handler for the specified func
func NewHandler[T any, PT MsgPtr[T]](fn func(ctx context.Context, m PT) error) TypeHandler {
	var z T
	m := (PT)(&z)
	return &typeHandler[PT]{
		mtype: MessageType(m),
		fn:    fn,
	}
}

// Handle handles the message
func (h HandlerFunc) Handle(ctx context.Context, m proto.Message) error {
	return h(ctx, m)
}

// Handle handles the message
func (h typeHandler[T]) Handle(ctx context.Context, m proto.Message) error {
	tm, ok := m.(T)
	if !ok {
		return NewPersistentError(fmt.Errorf("invalid type: got %T, expected %s", m, h.mtype))
	}
	return h.fn(ctx, tm)
}

// Type returns the message type
func (h typeHandler[T]) Type() string {
	return h.mtype
}

// MessageType returns the type for the supplied message
func MessageType(m proto.Message) string {
	return string(m.ProtoReflect().Descriptor().FullName())
}
