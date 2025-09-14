package schedule

//go:generate mockery --name Queue --output ./mocks --outpkg mocks --filename queue.go --structname Queue
//go:generate mockery --name QueueStats --output ./mocks --outpkg mocks --filename queue_stats.go --structname QueueStats
//go:generate mockery --name Logger --output ./mocks --outpkg mocks --filename logger.go --structname Logger

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// NewScheduler creates a new instance of *Scheduler with all dependencies.
func NewScheduler(
	checkPeriod time.Duration,
	log Logger,
	queue Queue,
	semaphore Semaphore,
) *Scheduler {
	return &Scheduler{
		checkPeriod: checkPeriod,
		log:         log,
		queue:       queue,
		semaphore:   semaphore,
	}
}

type (
	// Scheduler allows delayed execution of tasks at the specified time.
	Scheduler struct {
		checkPeriod time.Duration

		log       Logger
		queue     Queue
		semaphore Semaphore

		panics atomic.Uint64
	}

	// Queue is a priority queue of tasks
	Queue interface {
		Add(execAt time.Time, fn func(ctx context.Context)) error
		PopBatch(execAt time.Time) []func(ctx context.Context)
		Stats() QueueStats
	}

	// QueueStats stats of the queue
	QueueStats interface {
		Tasks() uint64
		EstimatedTaskSize() uint64
		UsedBytes() uint64
		UsedMiB() float64
		CapacityBytes() uint64
	}

	// Semaphore limits the number of concurrent tasks.
	Semaphore interface {
		Acquire(ctx context.Context, n int64) error
		Release(n int64)
	}

	// Logger writes error logs when panic occurs.
	Logger interface {
		Error(msg string)
	}
)

// Add adds a new task to the queue to be executed at the specified time.
func (s *Scheduler) Add(execAt time.Time, fn func(ctx context.Context)) error {
	if err := s.queue.Add(execAt, fn); err != nil {
		return fmt.Errorf("failed to add task to the queue: %w", err)
	}

	return nil
}

// Run executes tasks from the queue until the context is canceled.
// The only possible error is context.Canceled.
func (s *Scheduler) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case <-timer.C:
		}

		for _, runFunc := range s.queue.PopBatch(time.Now()) {
			if err := s.semaphore.Acquire(ctx, 1); err != nil {
				wg.Wait()
				return err
			}

			wg.Go(func() {
				defer func() {
					if rec := recover(); rec != nil {
						s.panics.Add(1)
						s.log.Error(fmt.Sprintf("panic at scheduled task: %v", rec))
					}

					s.semaphore.Release(1)
				}()

				runFunc(ctx)
			})
		}

		timer.Reset(s.checkPeriod)
	}
}

// Stats return a snapshot of the scheduler stats.
func (s *Scheduler) Stats() *Stats {
	queueStats := s.queue.Stats()

	return &Stats{
		QueueStats: queueStats,
		panics:     s.panics.Load(),
	}
}
