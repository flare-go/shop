package shop

import (
	"context"
	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"
)

type EventProcessor interface {
	ProcessEvent(ctx context.Context, event *stripe.Event) error
}

type WorkerPool struct {
	workers   chan struct{}
	tasks     chan func()
	logger    *zap.Logger
	processor EventProcessor
}

func NewWorkerPool(size int, processor EventProcessor, logger *zap.Logger) *WorkerPool {
	wp := &WorkerPool{
		workers:   make(chan struct{}, size),
		tasks:     make(chan func(), 1000),
		logger:    logger,
		processor: processor,
	}

	for i := 0; i < size; i++ {
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	for task := range wp.tasks {
		wp.workers <- struct{}{}
		task()
		<-wp.workers
	}
}

func (wp *WorkerPool) Submit(ctx context.Context, event *stripe.Event) {
	wp.tasks <- func() {
		if err := wp.processor.ProcessEvent(ctx, event); err != nil {
			wp.logger.Error("Failed to process event",
				zap.Error(err),
				zap.String("event_type", string(event.Type)),
				zap.String("event_id", event.ID))
		}
	}
}

func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)
	for i := 0; i < cap(wp.workers); i++ {
		<-wp.workers
	}
}
