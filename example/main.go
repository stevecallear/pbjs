package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stevecallear/pbjs"
	"github.com/stevecallear/pbjs/internal/proto/testpb"
)

var logger *slog.Logger

func main() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	const streamName = "ex_orders"
	if err = createStream(ctx, js, streamName); err != nil {
		log.Fatal(err)
	}

	djsc, err := createDispatchedConsumer(ctx, js, streamName, "ex_order_dispatched")
	if err != nil {
		log.Fatal(err)
	}

	fjsc, err := createFulfilledConsumer(ctx, js, streamName, "ex_order_fulfilled")
	if err != nil {
		log.Fatal(err)
	}

	pub := pbjs.NewPublisher(js,
		pbjs.WithSubjectConvention(pbjs.AnnotationSubjectConvention), // use the subject template defined in the proto definition
		pbjs.WithPublisherMiddleware(pbjs.LoggingMiddleware(logger.With("src", "publisher"))),
	)

	dcon, err := pbjs.NewConsumer(djsc,
		pbjs.NewHandler(func(ctx context.Context, m *testpb.OrderDispatchedEvent) error {
			// appropriate headers will be propagated from the handler context
			return pub.Publish(ctx, &testpb.OrderFulfilledEvent{Id: m.GetId()})
		}),
		pbjs.WithLogger(logger),
		pbjs.WithConsumerMiddleware(pbjs.LoggingMiddleware(logger.With("src", "dispatched_consumer"))),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer dcon.Close()

	fcon, err := pbjs.NewConsumer(fjsc,
		pbjs.NewHandler(func(ctx context.Context, m *testpb.OrderFulfilledEvent) error {
			logger.Info("order complete", "id", m.GetId())
			return nil
		}),
		pbjs.WithLogger(logger),
		pbjs.WithConsumerMiddleware(pbjs.LoggingMiddleware(logger.With("src", "fulfilled_consumer"))),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer fcon.Close()

	go publish(ctx, pub)

	<-ctx.Done()
}

func publish(ctx context.Context, p *pbjs.Publisher) {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			err := p.Publish(ctx, &testpb.OrderDispatchedEvent{Id: uuid.NewString()})
			if err != nil {
				log.Fatal(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func createStream(ctx context.Context, js jetstream.JetStream, stream string) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     stream,
		Subjects: []string{"ORDER.>"},
	})
	return err
}

func createDispatchedConsumer(ctx context.Context, js jetstream.JetStream, stream, consumer string) (jetstream.Consumer, error) {
	c, err := js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          consumer,
		Durable:       consumer,
		FilterSubject: "ORDER.*.dispatched",
	})
	return c, err
}

func createFulfilledConsumer(ctx context.Context, js jetstream.JetStream, stream, consumer string) (jetstream.Consumer, error) {
	c, err := js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          consumer,
		Durable:       consumer,
		FilterSubject: "ORDER.*.fulfilled",
	})
	return c, err
}
