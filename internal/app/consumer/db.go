package consumer

import (
	"context"
	"sync"
	"time"

	"github.com/hablof/logistic-package-api/internal/app/repo"
	"github.com/hablof/logistic-package-api/internal/model"
)

type Consumer interface {
	Start()
	Close()
}

type consumer struct {
	consumerCount uint64
	eventsChannel chan<- model.PackageEvent

	repo repo.EventRepo

	batchSize       uint64
	consumeInterval time.Duration

	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

type ConsumerConfig struct {
	ConsumeCount    uint64
	EventsChannel   chan<- model.PackageEvent
	Repo            repo.EventRepo
	BatchSize       uint64
	ConsumeInterval time.Duration
}

func NewDbConsumer(cfg ConsumerConfig) Consumer {
	wg := &sync.WaitGroup{}

	return &consumer{
		cancel:          func() {},
		consumerCount:   cfg.ConsumeCount,
		batchSize:       cfg.BatchSize,
		consumeInterval: cfg.ConsumeInterval,
		repo:            cfg.Repo,
		eventsChannel:   cfg.EventsChannel,
		wg:              wg,
	}
}

func (c *consumer) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	for i := uint64(0); i < c.consumerCount; i++ {
		c.wg.Add(1)

		go func() {
			defer c.wg.Done()
			ticker := time.NewTicker(c.consumeInterval)
			for {
				select {
				case <-ticker.C:
					// this case block not interrupted by ctx.Done(), so implements At-least-once
					events, err := c.repo.Lock(c.batchSize)
					if err != nil {
						continue
					}
					for _, event := range events {
						c.eventsChannel <- event
					}

				case <-ctx.Done():
					ticker.Stop()
					return
				}
			}
		}()
	}
}

func (c *consumer) Close() {
	c.cancel()
	c.wg.Wait()
}
