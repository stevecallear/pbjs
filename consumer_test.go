package pbjs_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"

	"github.com/stevecallear/pbjs"
	"github.com/stevecallear/pbjs/internal/proto/testpb"
)

func ExampleNewConsumer() {
	js, ok := getJS()
	if !ok {
		fmt.Println("value") // skip the example if jetstream is not available
		return
	}

	ctx := context.Background()

	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "example_stream",
		Subjects: []string{"testpb.>"},
	})
	if err != nil {
		log.Fatal(err)
	}

	jsc, err := js.CreateOrUpdateConsumer(ctx, "example_stream", jetstream.ConsumerConfig{
		Name:    "example_consumer",
		Durable: "example_consumer",
	})
	if err != nil {
		log.Fatal(err)
	}

	var value string
	ch := make(chan struct{})

	con, err := pbjs.NewConsumer(jsc,
		pbjs.NewHandler(func(ctx context.Context, in *testpb.MessageA) error {
			defer close(ch)
			value = in.GetValue()
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer con.Close()

	pub := pbjs.NewPublisher(js)
	err = pub.Publish(ctx, &testpb.MessageA{Value: "value"})
	if err != nil {
		log.Fatal(err)
	}

	select {
	case <-ch:
	case <-time.After(time.Second):
		log.Fatal("timeout waiting for channel")
	}

	fmt.Println(value)
	// output: value
}

func TestConsumer(t *testing.T) {
	js := requireJS(t)

	jss := must(newStream(t.Context(), js, newStreamConfig()))
	defer jss.close()

	t.Run("should handle the message", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig()))
		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention))

		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc, pbjs.NewHandler(func(ctx context.Context, in *testpb.MessageA) error {
			close(ch)
			return nil
		})))
		defer con.Close()

		err := pub.Publish(t.Context(), &testpb.MessageA{Value: "abc123"})
		if err != nil {
			t.Fatal(err)
		}

		wait(t, ch)
	})

	t.Run("should return persistent for resolution errors", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig()))

		var persistent bool
		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc, nil, pbjs.WithErrorHandler(func(ctx context.Context, m jetstream.Msg, err error) {
			defer close(ch)
			persistent = pbjs.IsPersistentError(err)
			m.Term()
		})))
		defer con.Close()

		err := rawPublish(t.Context(), js, jsc.subjectPrefix+".message", "invalid", []byte{})
		if err != nil {
			t.Fatal(err)
		}

		wait(t, ch)

		if !persistent {
			t.Error("got other error, expected persistent")
		}
	})

	t.Run("should return persistent for unmarshal errors", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig()))

		var persistent bool
		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc, nil, pbjs.WithErrorHandler(func(ctx context.Context, m jetstream.Msg, err error) {
			defer close(ch)
			persistent = true
			m.Term()
		})))
		defer con.Close()

		mt := pbjs.MessageType(new(testpb.MessageA))
		err := rawPublish(t.Context(), js, jsc.subjectPrefix+"."+mt, mt, []byte("%^&"))
		if err != nil {
			t.Fatal(err)
		}

		wait(t, ch)

		if !persistent {
			t.Error("got other error, expected persistent")
		}
	})

	t.Run("should return handler errors", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig()))

		var errstr string
		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc,
			pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
				return errors.New("error")
			}),
			pbjs.WithErrorHandler(func(ctx context.Context, m jetstream.Msg, err error) {
				defer close(ch)
				if err != nil {
					errstr = err.Error()
				}
				m.Term()
			}),
		))
		defer con.Close()

		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention))
		err := pub.Publish(t.Context(), new(testpb.MessageA))
		if err != nil {
			t.Fatal(err)
		}

		wait(t, ch)

		if act, exp := errstr, "error"; act != exp {
			t.Errorf("got %s, expected %s", act, exp)
		}
	})

	t.Run("should retry on transient error", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig(func(c *jetstream.ConsumerConfig) {
			c.MaxDeliver = 2
		})))

		var n int32
		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc, pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
			if atomic.AddInt32(&n, 1) >= 2 {
				close(ch)
			}
			return errors.New("error") // transient error
		})))
		defer con.Close()

		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention))
		if err := pub.Publish(t.Context(), new(testpb.MessageA)); err != nil {
			log.Fatal(err)
		}

		wait(t, ch)

		if act, exp := atomic.LoadInt32(&n), int32(2); act != exp {
			t.Errorf("got %d invocations, expected %d", act, exp)
		}
	})

	t.Run("should terminate on persistent error", func(t *testing.T) {
		jsc := must(jss.newConsumer(newConsumerConfig(func(c *jetstream.ConsumerConfig) {
			c.MaxDeliver = 2
		})))

		var n int32
		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc, pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
			atomic.AddInt32(&n, 1)
			go func() {
				defer close(ch)
				<-time.After(100 * time.Millisecond)
			}()
			return pbjs.NewPersistentError(errors.New("error"))
		})))
		defer con.Close()

		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention))
		if err := pub.Publish(t.Context(), new(testpb.MessageA)); err != nil {
			log.Fatal(err)
		}

		wait(t, ch)

		if act, exp := atomic.LoadInt32(&n), int32(1); act != exp {
			t.Errorf("got %d invocations, expected %d", act, exp)
		}
	})
}

func TestWithLogger(t *testing.T) {
	js := requireJS(t)

	jss := must(newStream(t.Context(), js, newStreamConfig()))
	defer jss.close()

	t.Run("should log errors", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		l := slog.New(slog.NewTextHandler(b, nil))

		jsc := must(jss.newConsumer(newConsumerConfig()))

		ch := make(chan struct{})
		con, err := pbjs.NewConsumer(jsc,
			pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
				return pbjs.NewPersistentError(errors.New("error"))
			}),
			pbjs.WithLogger(l),
			pbjs.WithErrorHandler(func(ctx context.Context, m jetstream.Msg, err error) {
				defer close(ch)
				m.Term()
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer con.Close()

		pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention))
		if err = pub.Publish(t.Context(), new(testpb.MessageA)); err != nil {
			t.Fatal(err)
		}

		wait(t, ch)

		if s := b.String(); !strings.Contains(s, "err=error") {
			t.Errorf("got %s expected error message", s)
		}
	})
}

func TestWithConsumerMiddleware(t *testing.T) {
	js := requireJS(t)

	t.Run("should use the middleware", func(t *testing.T) {
		jss := must(newStream(t.Context(), js, newStreamConfig()))
		defer jss.close()

		ok := new(atomic.Bool)

		jsc := must(jss.newConsumer(newConsumerConfig()))

		ch := make(chan struct{})
		con := must(pbjs.NewConsumer(jsc,
			pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
				defer close(ch)
				return nil
			}),
			pbjs.WithConsumerMiddleware(pbjs.MiddlewareFunc(func(h pbjs.Handler) pbjs.Handler {
				return pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
					ok.Store(true)
					return h.Handle(ctx, m)
				})
			})),
		))
		defer con.Close()

		err := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(jsc.subjectConvention)).
			Publish(t.Context(), new(testpb.MessageA))
		if err != nil {
			t.Fatal(err)
		}

		wait(t, ch)

		if act, exp := ok.Load(), true; act != exp {
			t.Errorf("got %v, expected %v", act, exp)
		}
	})
}
