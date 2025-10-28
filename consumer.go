package pbjs

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type (
	// Consumer represents a JetStream consumer
	Consumer struct {
		jsc     jetstream.ConsumeContext
		handler Handler
		timeout time.Duration
		errorFn ErrorFunc
		logger  *slog.Logger
	}

	// ConsumerOptions represents consumer options
	ConsumerOptions struct {
		Middleware []MiddlewareFunc
		Logger     *slog.Logger
		Timeout    time.Duration
		ErrorFn    ErrorFunc
	}

	// ErrorFunc represents a consumer error handler.
	// By default all persistent errors result in Term, while all others result in Nak.
	ErrorFunc func(ctx context.Context, m jetstream.Msg, err error)
)

var defaultOptions = ConsumerOptions{
	Logger:  slog.New(slog.DiscardHandler),
	Timeout: 5 * time.Second,
	ErrorFn: func(ctx context.Context, m jetstream.Msg, err error) {
		if IsPersistentError(err) {
			m.Term()
			return
		}
		m.Nak()
	},
}

// WithLogger specifies the consumer logger
func WithLogger(l *slog.Logger) func(*ConsumerOptions) {
	return func(o *ConsumerOptions) {
		o.Logger = l
	}
}

// WithErrorHandler specifies the error handler func
func WithErrorHandler(fn ErrorFunc) func(*ConsumerOptions) {
	return func(o *ConsumerOptions) {
		o.ErrorFn = fn
	}
}

// WithConsumerMiddleware specifies the consumer middleware
func WithConsumerMiddleware(m ...MiddlewareFunc) func(*ConsumerOptions) {
	return func(o *ConsumerOptions) {
		o.Middleware = m
	}
}

// NewConsumer returns a new consumer
func NewConsumer(jc jetstream.Consumer, h Handler, optFns ...func(*ConsumerOptions)) (*Consumer, error) {
	o := defaultOptions
	for _, fn := range optFns {
		fn(&o)
	}

	for i := len(o.Middleware) - 1; i >= 0; i-- {
		h = o.Middleware[i](h)
	}

	c := &Consumer{
		handler: h,
		timeout: o.Timeout,
		errorFn: o.ErrorFn,
		logger:  o.Logger,
	}

	var err error
	c.jsc, err = jc.Consume(c.handle)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Close drains the underlying [jetstream.ConsumeContext]
func (c *Consumer) Close() error {
	c.jsc.Drain()
	<-c.jsc.Closed()
	return nil
}

func (c *Consumer) handle(m jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	hd := m.Headers()
	hd.Set(HdrSubject, m.Subject())
	ctx = SetContextHeader(ctx, hd)

	if md, err := m.Metadata(); err == nil {
		ctx = SetContextMetadata(ctx, md)
	}

	mt := hd.Get(HdrMsgType)
	logger := c.logger.With(slog.String(HdrNatsMsgID, hd.Get(HdrNatsMsgID)), slog.String(HdrMsgType, mt))

	pm, err := resolveMessage(mt)
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error resolving message", slog.String("err", err.Error()))
		c.errorFn(ctx, m, NewPersistentError(err))
		return
	}

	if err = proto.Unmarshal(m.Data(), pm); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error unmarshaling message", slog.String("err", err.Error()))
		c.errorFn(ctx, m, NewPersistentError(err))
		return
	}

	err = c.handler.Handle(ctx, pm)
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error handling message", slog.String("err", err.Error()))
		c.errorFn(ctx, m, err)
		return
	}

	m.Ack()
}

func resolveMessage(typ string) (proto.Message, error) {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typ))
	if err != nil {
		return nil, err
	}
	return mt.New().Interface(), nil
}
