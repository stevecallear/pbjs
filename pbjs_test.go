package pbjs_test

import (
	"context"
	"errors"
	"log"
	"math/rand/v2"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"

	"github.com/stevecallear/pbjs"
)

var (
	js    jetstream.JetStream
	hasJS = new(atomic.Bool)
)

func TestMain(m *testing.M) {
	code := testMain(m)
	os.Exit(code)
}

func testMain(m *testing.M) int {
	if nu := os.Getenv("NATS_URL"); nu != "" {
		nc, err := nats.Connect(nu)
		if err != nil {
			panic(err)
		}
		defer nc.Close()

		js, err = jetstream.New(nc)
		if err != nil {
			panic(err)
		}

		hasJS.Store(true)
	}
	return m.Run()
}

type (
	stream struct {
		js        jetstream.JetStream
		ctx       context.Context
		name      string
		consumers []*consumer
	}

	consumer struct {
		name          string
		subjectPrefix string
		jetstream.Consumer
	}
)

func requireJS(t *testing.T) jetstream.JetStream {
	t.Helper()
	if !hasJS.Load() {
		t.Skip("jetstream not initialised")
	}
	return js
}

func getJS() (jetstream.JetStream, bool) {
	return js, hasJS.Load()
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func wait(t *testing.T, ch <-chan struct{}) {
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Errorf("timeout waiting for channel")
	}
}

func newStreamConfig(fns ...func(*jetstream.StreamConfig)) jetstream.StreamConfig {
	n := newID()
	cfg := jetstream.StreamConfig{
		Name:     n,
		Subjects: []string{n + ".>"},
	}
	for _, fn := range fns {
		fn(&cfg)
	}
	return cfg
}

func newConsumerConfig(fns ...func(*jetstream.ConsumerConfig)) jetstream.ConsumerConfig {
	n := newID()
	cfg := jetstream.ConsumerConfig{
		Name:          n,
		Durable:       n,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	}
	for _, fn := range fns {
		fn(&cfg)
	}
	return cfg
}

func newStream(ctx context.Context, js jetstream.JetStream, cfg jetstream.StreamConfig) (*stream, error) {
	_, err := js.CreateStream(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &stream{
		js:   js,
		ctx:  ctx,
		name: cfg.Name,
	}, nil
}

func (s *stream) newConsumer(cfg jetstream.ConsumerConfig) (*consumer, error) {
	pre := s.name + "." + cfg.Name
	log.Println(pre)
	cfg.FilterSubject = pre + ".>"
	cfg.FilterSubjects = nil

	c, err := js.CreateConsumer(s.ctx, s.name, cfg)
	if err != nil {
		return nil, err
	}

	wc := &consumer{
		name:          cfg.Name,
		subjectPrefix: pre,
		Consumer:      c,
	}

	s.consumers = append(s.consumers, wc)
	return wc, nil
}

func (s *stream) close() (err error) {
	var derr error
	for _, c := range s.consumers {
		if derr = s.js.DeleteConsumer(s.ctx, s.name, c.name); derr != nil {
			err = errors.Join(err, derr)
		}
	}

	if derr = s.js.DeleteStream(s.ctx, s.name); derr != nil {
		err = errors.Join(err, derr)
	}

	return err
}

func (c *consumer) subjectConvention(ctx context.Context, m proto.Message) (string, error) {
	return c.subjectPrefix + "." + pbjs.MessageType(m), nil
}

func rawPublish(ctx context.Context, js jetstream.JetStream, subject, msgtype string, data []byte) error {
	nm := nats.NewMsg(subject)
	nm.Data = data
	nm.Header.Set(pbjs.HdrNatsMsgID, uuid.NewString())
	nm.Header.Set(pbjs.HdrSubject, subject)
	nm.Header.Set(pbjs.HdrMsgType, msgtype)

	_, err := js.PublishMsg(ctx, nm)
	return err
}

func newID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var s string
	for range 8 {
		s += string(alphabet[rand.IntN(len(alphabet))])
	}
	return s
}
