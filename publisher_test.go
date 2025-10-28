package pbjs_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/stevecallear/pbjs"
	"github.com/stevecallear/pbjs/internal/proto/testpb"
)

func TestPublisher_Publish(t *testing.T) {
	js := requireJS(t)

	jss := must(newStream(t.Context(), js, newStreamConfig()))
	defer jss.close()

	t.Run("should publish the message", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig()))
		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention))

		err := pub.Publish(t.Context(), &testpb.OrderDispatchedEvent{})
		if err != nil {
			t.Errorf("got %v, expected nil", err)
		}
	})

	t.Run("should return an error if the subject convention fails", func(t *testing.T) {
		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(func(ctx context.Context, m proto.Message) (string, error) {
			return "", errors.New("error")
		}))

		err := pub.Publish(t.Context(), &testpb.OrderDispatchedEvent{})
		if err == nil {
			t.Error("got nil, expected error")
		}
	})

	t.Run("should remove expect headers", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig()))

		var n int32
		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc, pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
			defer close(ch)
			hd := pbjs.GetContextHeader(ctx)
			for k := range hd {
				if strings.HasPrefix(k, "Nats-Expected-") {
					atomic.AddInt32(&n, 1)
				}
			}
			return nil
		})))
		defer con.Close()

		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention))

		hd := make(nats.Header)
		hd.Set("Nats-Expected-Test", "value")
		ctx := pbjs.SetContextHeader(t.Context(), hd)

		err := pub.Publish(ctx, new(testpb.OrderDispatchedEvent))
		if err != nil {
			t.Fatal(err)
		}

		wait(t, ch)

		if act, exp := atomic.LoadInt32(&n), int32(0); act != exp {
			t.Errorf("got %d headers, expected %d", act, exp)
		}
	})
}

func TestWithPublisherMiddleware(t *testing.T) {
	js := requireJS(t)

	jss := must(newStream(t.Context(), js, newStreamConfig()))
	defer jss.close()

	t.Run("should apply the middleware", func(t *testing.T) {
		con := must(jss.newConsumer(newConsumerConfig()))

		var count int32

		pub := pbjs.NewPublisher(js,
			pbjs.WithSubjectConvention(con.subjectConvention),
			pbjs.WithPublisherMiddleware(func(h pbjs.Handler) pbjs.Handler {
				return pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
					atomic.AddInt32(&count, 1)
					return h.Handle(ctx, m)
				})
			}),
		)

		err := pub.Publish(t.Context(), &testpb.OrderDispatchedEvent{})
		if err != nil {
			t.Errorf("got %v, expected nil", err)
		}

		if act, exp := atomic.LoadInt32(&count), int32(1); act != exp {
			t.Errorf("got %d messages, expected %d", act, exp)
		}
	})
}
