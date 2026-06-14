package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	chatdemoruntime "github.com/go-go-golems/sessionstream/examples/goja-redis-chatdemo-server/internal/xgojaruntime"
	ssprovider "github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream/provider"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	redisAddr := envOrDefault("REDIS_ADDR", "127.0.0.1:6379")
	topic := envOrDefault("SESSIONSTREAM_BUS_TOPIC", "sessionstream.chatdemo.redis")
	consumerID := envOrDefault("SESSIONSTREAM_CONSUMER_ID", fmt.Sprintf("redis-chatdemo-%d", os.Getpid()))

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis %s: %w", redisAddr, err)
	}
	defer func() { _ = redisClient.Close() }()

	logger := watermill.NopLogger{}
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: redisClient}, logger)
	if err != nil {
		return fmt.Errorf("create redisstream publisher: %w", err)
	}
	defer func() { _ = publisher.Close() }()

	// Use fan-out mode (empty ConsumerGroup) so every server process receives every
	// event and can fan it out to its own websocket clients. A shared ConsumerGroup
	// would turn this into work-queue semantics where only one process handles each
	// event.
	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:                        redisClient,
		Consumer:                      consumerID,
		ConsumerGroup:                 "",
		FanOutOldestId:                "$",
		BlockTime:                     time.Second,
		DisableIndefiniteInitialBlock: true,
		ShouldStopOnReadErrors:        func(error) bool { return false },
		CheckConsumersInterval:        30 * time.Second,
		NackResendSleep:               time.Second,
		ClaimInterval:                 30 * time.Second,
		ClaimBatchSize:                32,
		MaxIdleTime:                   time.Minute,
		ConsumerTimeout:               5 * time.Minute,
	}, logger)
	if err != nil {
		return fmt.Errorf("create redisstream subscriber: %w", err)
	}
	defer func() { _ = subscriber.Close() }()

	bundle, err := chatdemoruntime.NewBundle(chatdemoruntime.Options{
		Out: os.Stdout,
		ConfigureServices: func(services *app.HostServices) {
			_ = services.SetHostService(ssprovider.HostServiceKey, ssprovider.HostOptions{
				HubOptions: []ss.HubOption{
					ss.WithEventBus(publisher, subscriber, ss.WithBusTopic(topic)),
				},
			})
		},
	})
	if err != nil {
		return fmt.Errorf("create xgoja runtime bundle: %w", err)
	}

	root := &cobra.Command{
		Use:   "redis-chatdemo",
		Short: "xgoja chatdemo host wired to Redis-backed Watermill events",
	}
	bundle.AttachDefaultCommands(root)
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
