package pbjs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

type (
	// Publisher represents a JetStream publisher
	Publisher struct {
		js                jetstream.JetStream
		handler           Handler
		subjectConvention SubjectConventionFunc
	}

	// PublisherOptions represents publisher options
	PublisherOptions struct {
		Middleware        []MiddlewareFunc
		SubjectConvention SubjectConventionFunc
	}

	// SubjectConventionFunc represents a subjection convention func.
	// By default publishers use the value of [MessageType] as the subject.
	SubjectConventionFunc func(ctx context.Context, m proto.Message) (string, error)
)

var defaultPublisherOptions = PublisherOptions{
	SubjectConvention: func(_ context.Context, m proto.Message) (string, error) {
		return MessageType(m), nil
	},
}

// WithSubjectConvention specifies the publisher subject convention
func WithSubjectConvention(fn SubjectConventionFunc) func(o *PublisherOptions) {
	return func(o *PublisherOptions) {
		o.SubjectConvention = fn
	}
}

// WithPublisherMiddleware specifies the publisher middleware
func WithPublisherMiddleware(m ...MiddlewareFunc) func(o *PublisherOptions) {
	return func(o *PublisherOptions) {
		o.Middleware = append(o.Middleware, m...)
	}
}

// NewPublisher returns a new publisher
func NewPublisher(js jetstream.JetStream, optFns ...func(*PublisherOptions)) *Publisher {
	o := defaultPublisherOptions
	for _, fn := range optFns {
		fn(&o)
	}

	p := &Publisher{
		js:                js,
		subjectConvention: o.SubjectConvention,
	}

	p.handler = ApplyMiddleware(HandlerFunc(p.handle), o.Middleware...)
	return p
}

// Publish publishes the message
func (p *Publisher) Publish(ctx context.Context, m proto.Message) error {
	h := GetContextHeader(ctx)
	removeExpectHeaders(h)

	h.Set(HdrNatsMsgID, uuid.NewString())
	h.Set(HdrNatsTimestamp, time.Now().UTC().Format(time.RFC3339))
	h.Set(HdrMsgType, MessageType(m))
	h.Set(HdrContentType, mimeTypeProtobuf)

	if h.Get(HdrCorrelationID) == "" {
		h.Set(HdrCorrelationID, uuid.NewString())
	}

	ctx = SetContextHeader(ctx, h)
	return p.handler.Handle(ctx, m)
}

func (p *Publisher) handle(ctx context.Context, m proto.Message) error {
	sub, err := p.subjectConvention(ctx, m)
	if err != nil {
		return err
	}

	nm := nats.NewMsg(sub)
	nm.Header = GetContextHeader(ctx)
	nm.Data, err = proto.Marshal(m)
	if err != nil {
		return err
	}

	_, err = p.js.PublishMsg(ctx, nm)
	return err
}
