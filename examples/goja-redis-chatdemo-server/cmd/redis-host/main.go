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
	bridge := &redisBridge{settings: redisHostSettings{
		RedisAddr:  "127.0.0.1:6379",
		BusTopic:   "sessionstream.chatdemo.redis",
		ConsumerID: fmt.Sprintf("redis-chatdemo-%d", os.Getpid()),
	}}
	defer bridge.Close()

	bundle, err := chatdemoruntime.NewBundle(chatdemoruntime.Options{
		Out: os.Stdout,
		ConfigureServices: func(services *app.HostServices) {
			_ = services.SetHostService(ssprovider.HostServiceKey, ssprovider.HostOptions{
				HubOptions: []ss.HubOption{bridge.HubOption()},
			})
		},
	})
	if err != nil {
		return fmt.Errorf("create xgoja runtime bundle: %w", err)
	}

	root := &cobra.Command{
		Use:   "redis-chatdemo",
		Short: "xgoja chatdemo host wired to Redis-backed Watermill events",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return bridge.Open(cmd.Context())
		},
	}
	bundle.AttachDefaultCommands(root)
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

type redisHostSettings struct {
	RedisAddr  string
	BusTopic   string
	ConsumerID string
}

type redisBridge struct {
	settings    redisHostSettings
	redisClient *redis.Client
	publisher   *redisstream.Publisher
	subscriber  *redisstream.Subscriber
}

func (b *redisBridge) Open(ctx context.Context) error {
	if b == nil {
		return fmt.Errorf("redis bridge is not configured")
	}
	if b.publisher != nil && b.subscriber != nil {
		return nil
	}
	redisClient := redis.NewClient(&redis.Options{Addr: b.settings.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		return fmt.Errorf("ping redis %s: %w", b.settings.RedisAddr, err)
	}

	logger := watermill.NopLogger{}
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: redisClient}, logger)
	if err != nil {
		_ = redisClient.Close()
		return fmt.Errorf("create redisstream publisher: %w", err)
	}

	// Use fan-out mode (empty ConsumerGroup) so every server process receives every
	// event and can fan it out to its own websocket clients. A shared ConsumerGroup
	// would turn this into work-queue semantics where only one process handles each
	// event.
	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:                        redisClient,
		Consumer:                      b.settings.ConsumerID,
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
		_ = publisher.Close()
		_ = redisClient.Close()
		return fmt.Errorf("create redisstream subscriber: %w", err)
	}

	b.redisClient = redisClient
	b.publisher = publisher
	b.subscriber = subscriber
	return nil
}

func (b *redisBridge) HubOption() ss.HubOption {
	return func(h *ss.Hub) error {
		if b == nil || b.publisher == nil || b.subscriber == nil {
			return fmt.Errorf("redis bridge is not open")
		}
		return ss.WithEventBus(b.publisher, b.subscriber, ss.WithBusTopic(b.settings.BusTopic))(h)
	}
}

func (b *redisBridge) Close() {
	if b == nil {
		return
	}
	if b.subscriber != nil {
		_ = b.subscriber.Close()
	}
	if b.publisher != nil {
		_ = b.publisher.Close()
	}
	if b.redisClient != nil {
		_ = b.redisClient.Close()
	}
}
