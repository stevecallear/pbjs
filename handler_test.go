package pbjs_test

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/stevecallear/pbjs"
	"github.com/stevecallear/pbjs/internal/proto/testpb"
)

func TestNewHandler(t *testing.T) {
	t.Run("should return the type handler", func(t *testing.T) {
		var invoked bool
		sut := pbjs.NewHandler(func(ctx context.Context, in *testpb.OrderDispatchedEvent) error {
			invoked = true
			return nil
		})

		err := sut.Handle(t.Context(), &testpb.OrderDispatchedEvent{
			Id: "abc123",
		})
		if err != nil {
			t.Errorf("got %v, expected nil", err)
		}
		if !invoked {
			t.Errorf("got false, expected true")
		}
	})

	t.Run("should return an error if invoked with invalid type", func(t *testing.T) {
		sut := pbjs.NewHandler(func(ctx context.Context, in *testpb.OrderDispatchedEvent) error {
			return nil
		})

		err := sut.Handle(t.Context(), new(testpb.OrderFulfilledEvent))
		if err == nil {
			t.Error("got nil, expected error")
		}
		if !pbjs.IsPersistentError(err) {
			t.Errorf("got %v, expected persistent error", err)
		}
	})
}

func TestHandlerFunc_Handle(t *testing.T) {
	t.Run("should invoke the func", func(t *testing.T) {
		var invoked bool
		sut := pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
			invoked = true
			return nil
		})

		err := sut.Handle(t.Context(), nil)
		if err != nil {
			t.Errorf("got %v, expected nil", err)
		}
		if !invoked {
			t.Errorf("got false, expected true")
		}
	})
}

func TestMessageType(t *testing.T) {
	t.Run("should return the message type", func(t *testing.T) {
		const exp = "testpb.OrderDispatchedEvent"
		act := pbjs.MessageType(new(testpb.OrderDispatchedEvent))
		if act != exp {
			t.Errorf("got %s, expected %s", act, exp)
		}
	})
}
