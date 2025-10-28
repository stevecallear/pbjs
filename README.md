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
err = pub.Publish(ctx, &testpb.OrderDispatchedEvent{Id: "abc123"})
if err != nil {
    log.Fatal(err)
}
```

## Handlers
`Consumers` accept a `Handler` implementation. `HandlerFunc` provides a convenience function to simplify handler creation. To avoid casts, `NewHandler` accepts a typed handler function and returns a `TypeHandler`.

## Middleware
Both `Publisher` and `Consumer` accept middleware on creation. A simple logging middleware is provided:
```
pub := pbjs.NewPublisher(js, pbjs.WithPublisherMiddleware(pbjs.LoggingMiddleware(logger)))
```

## Subject Convention
`Publisher` accepts a `SubjectConventionFunc` that allows the subject to be defined based on the publisher context and message. By default this simply returns the message type, but it can be customised for more complex situations:
```
pub := pbjs.NewPublisher(js, pbjs.WithSubjectConvention(func(ctx context.Context, m proto.Message) (string, error) {
    // convention logic
}))
```
A future goal of the module is to provide a `protoc-gen` implementation to allow subject conventions to be defined as part of the `.proto` definition.

## Handler Errors
Message acknowledgement is based on the handler result. Handlers can use `NewError` to distinguish between transient and persistent errors:
```
h := pbjs.HandlerFunc(func(ctx context.Context, m *testpb.MessageB) error {
    state, err := store.Get(ctx, m.GetId())
    if err != nil {
        return pbjs.NewError(err, pbjs.ErrorTypeTransient)
    }

    seq := pbjs.GetContextMetadata(ctx).Sequence.Stream
    if state.Version > seq {
        return pbjs.NewError(errors.New("stale message", pbjs.ErrorTypePersistent))
    }
    
    // logic

    return nil
})
```
By default transient errors result in message `Nak` while persistent errors result in message `Term`. This behaviour can be customised using the `WithErrorHandler` option on consumer creation.

## Header Propagation
NATS headers and message metadata are stored in the publish/consume contexts. This allows header propagation when publishing from a consumer handler. By default `HdrCorrelationID` is not overriden in this scenario. All `Nats-Expect-*` headers are removed, however, to avoid unexpected behaviour.