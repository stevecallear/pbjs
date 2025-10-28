# pbjs
[![build](https://github.com/stevecallear/pbjs/actions/workflows/build.yml/badge.svg)](https://github.com/stevecallear/pbjs/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/stevecallear/pbjs/graph/badge.svg?token=3JBUN06BOD)](https://codecov.io/gh/stevecallear/pbjs)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/pbjs)](https://goreportcard.com/report/github.com/stevecallear/pbjs)

`pbjs` provides an opinionated wrapper for publishing/consuming Protobuf messages with NATS JetStream. It was created with the aim of bootstrapping stream interactions for a number of projects in a more lightweight manner than [Watermill](https://watermill.io/). That said, for production use I would recommend `watermill-nats` with a `protobuf` marshaler.

## Getting Started
```
go get github.com/stevecallear/pbjs@latest
```
```
// jetstream client, stream and consumer have been created prior
jsc, err := js.Consumer(ctx, "mystream", "myconsumer")
if err != nil {
    log.Fatal(err)
}

con, err := pbjs.NewConsumer(jsc, pbjs.NewHandler(func(ctx context.Context, m *testpb.OrderDispatchedEvent) error {
    log.Println(m.GetValue())
    return nil
}))
if err != nil {
    log.Fatal(err)
}
defer con.Close()

pub := pbjs.NewPublisher(js)
err = pub.Publish(ctx, &testpb.OrderDispatchedEvent{Id: uuid.NewString()})
if err != nil {
    log.Fatal(err)
}
```

## Subject Convention
`Publisher` accepts a `SubjectConventionFunc` that allows the subject to be defined based on the publisher context and message. By default this simply returns the message type, but it can be customised for more complex situations:
```
pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(func(ctx context.Context, m proto.Message) (string, error) {
    // convention logic
}))
```

### Message Annotations
It is possible to specify a subject as a message annotation to avoid complex convention logic. The subject can optionally include templated elements referring to the proto fields.
```
message OrderDispatchedEvent {
    option (pbjs.message).subject = "ORDER.{id}.dispatched";
    string id = 1;
}
```
```
pub := pbjs.NewPublisher(js, 
    pbjs.WithSubjectConvention(pbjs.AnnotationSubjectConvention),
    pbjs.WithMiddleware(ValidationMiddleware()), // if message fields are used for the subject convention, they should be validated prior    
)

pub.Publish(ctx, &testpb.OrderDispatchedEvent{Id: "abc123"})
// subject: ORDER.abc123.dispatched
```
The implementation currently uses `ProtoReflect` to extract the subject template and apply the field values. This can result in runtime errors due to configuration so unit testing is recommended to ensure correct outputs.

## Consumers
Consumers accept a `Handler` implementation. `HandlerFunc` provides a convenience function to simplify creation. To avoid casts, `NewHandler` accepts a typed handler function.

### Error Handling
Message acknowledgement can be controlled based on the handler error. By default nil errors result in `Ack`, errors wrapped using `NewPersistentError` result in `Term` and all other errors result in `Nak`.
```
h := pbjs.HandlerFunc(func(ctx context.Context, m *testpb.MessageB) error {
    state, err := store.Get(ctx, m.GetId())
    if err != nil {
        return err // transient error, re-deliver
    }

    seq := pbjs.GetContextMetadata(ctx).Sequence.Stream
    if state.Version > seq {
        return pbjs.NewPersistentError(errors.New("stale message"))
    }
    
    // logic

    return nil
})
```
The default behaviour can be customised using the `WithErrorHandler` option on consumer creation.

Message resolution and unmarshaling both result in persistent errors. Both errors occur outside of the handler chain and are therefore not addressable by middleware. Such errors are logged to `slog.Default()` by default with logger customisation achievable using `WithLogger` on consumer creation.

## Middleware
Both `Publisher` and `Consumer` accept middleware on creation. A simple logging middleware is provided:
```
pub := pbjs.NewPublisher(js, pbjs.WithPublisherMiddleware(pbjs.LoggingMiddleware(logger)))
```

### Validation
Validation middleware can be used to prevent the publish of invalid messages or to allow consumers to immediately reject invalid messages. A middleware function to achieve this with `protovalidate` would look like the following.
```
func ValidationMiddleware() MiddlewareFunc {
	return func(h Handler) Handler {
		return HandlerFunc(func(ctx context.Context, m proto.Message) error {
			if err := protovalidate.Validate(m); err != nil {
				return pbjs.NewError(err, pbjs.ErrTypePersistent) // return persistent to avoid re-delivery in consumers
			}
			return h.Handle(ctx, m)
		})
	}
}
```

## Header Propagation
NATS headers and message metadata are stored in the publish/consume contexts. This allows header propagation when publishing from a consumer handler. By default `HdrCorrelationID` is not overriden in this scenario. All `Nats-Expect-*` headers are removed, however, to avoid unexpected behaviour.