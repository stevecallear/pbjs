package pbjs_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/stevecallear/pbjs"
	"github.com/stevecallear/pbjs/internal/proto/testpb"
)

func TestLoggingMiddleware(t *testing.T) {
	js := requireJS(t)

	jss := must(newStream(t.Context(), js, newStreamConfig()))
	defer jss.close()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "should log success",
			err:  nil,
		},
		{
			name: "should log error",
			err:  errors.New("error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsc := must(jss.newConsumer(newConsumerConfig()))

			cb := newSyncBuffer()
			cl := slog.New(slog.NewJSONHandler(cb, &slog.HandlerOptions{Level: slog.LevelDebug}))

			ch := make(chan struct{})
			con := must(pbjs.NewConsumer(jsc,
				pbjs.HandlerFunc(func(ctx context.Context, m proto.Message) error {
					go func() {
						defer close(ch)
						<-time.After(100 * time.Millisecond)
					}()
					return pbjs.NewPersistentError(tt.err) // prevent redelivery
				}),
				pbjs.WithConsumerMiddleware(pbjs.LoggingMiddleware(cl)),
			))
			defer con.Close()

			pb := newSyncBuffer()
			pl := slog.New(slog.NewJSONHandler(pb, &slog.HandlerOptions{Level: slog.LevelDebug}))

			pub := pbjs.NewPublisher(js,
				pbjs.WithSubjectConvention(jsc.subjectConvention),
				pbjs.WithPublisherMiddleware(pbjs.LoggingMiddleware(pl)),
			)
			err := pub.Publish(t.Context(), new(testpb.OrderDispatchedEvent))
			if err != nil {
				log.Fatal(err)
			}

			wait(t, ch)

			if act, exp := len(strings.Split(strings.TrimSpace(cb.String()), "\n")), 2; act != exp {
				t.Errorf("got %d consumer log entries, expected %d", act, exp)
			}

			if act, exp := len(strings.Split(strings.TrimSpace(pb.String()), "\n")), 2; act != exp {
				t.Errorf("got %d publisher log entries, expected %d", act, exp)
			}
		})
	}
}

type syncBuffer struct {
	b  *bytes.Buffer
	mu sync.Mutex
}

func newSyncBuffer() *syncBuffer {
	return &syncBuffer{b: bytes.NewBuffer(nil)}
}

func (w *syncBuffer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *syncBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}
