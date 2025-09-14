package schedule_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/semaphore"

	"github.com/riabininkf/schedule"
	"github.com/riabininkf/schedule/mocks"
)

func TestScheduler_Add(t *testing.T) {
	t.Run("failed to add task to the queue", func(t *testing.T) {
		execAt := time.Now()

		queue := mocks.NewQueue(t)
		queue.On("Add", execAt, mock.AnythingOfType("func(context.Context)")).
			Return(assert.AnError)

		scheduler := schedule.NewScheduler(0, mocks.NewLogger(t), queue, semaphore.NewWeighted(1))
		err := scheduler.Add(execAt, func(ctx context.Context) {})

		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("positive case", func(t *testing.T) {
		execAt := time.Now()

		queue := mocks.NewQueue(t)
		queue.On("Add", execAt, mock.AnythingOfType("func(context.Context)")).
			Return(nil)

		scheduler := schedule.NewScheduler(0, mocks.NewLogger(t), queue, semaphore.NewWeighted(1))
		err := scheduler.Add(execAt, func(ctx context.Context) {})

		assert.NoError(t, err)
	})
}

func TestScheduler_Run(t *testing.T) {
	t.Run("2 workers take 2 tasks in 1 tick", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		var (
			firstTaskCalled  = make(chan struct{})
			secondTaskCalled = make(chan struct{})
		)

		queue := mocks.NewQueue(t)
		queue.On("PopBatch", mock.AnythingOfType("time.Time")).
			Return([]func(context.Context){
				func(ctx context.Context) {
					close(firstTaskCalled)
					<-ctx.Done() // ensures that both tasks are executed simultaneously
				},
				func(ctx context.Context) {
					close(secondTaskCalled)
					<-ctx.Done() // ensures that both tasks are executed simultaneously
				},
			}).Once()

		var wg sync.WaitGroup
		wg.Go(func() {
			// time.Hour guarantees that the scheduler will run only for 1 tick
			scheduler := schedule.NewScheduler(time.Hour, mocks.NewLogger(t), queue, semaphore.NewWeighted(2))

			err := scheduler.Run(ctx)
			assert.ErrorIs(t, err, context.Canceled)
		})

		select {
		case <-firstTaskCalled:
			<-secondTaskCalled
		case <-time.After(time.Second):
			t.Fatal("timeout: both tasks must be called within 1 second")
		}

		cancel()
		wg.Wait()
	})

	t.Run("panic in the task", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		queue := mocks.NewQueue(t)
		queue.On("PopBatch", mock.AnythingOfType("time.Time")).
			Return([]func(context.Context){func(ctx context.Context) {
				defer cancel()

				panic(assert.AnError)
			}}).Once()

		stats := mocks.NewQueueStats(t)
		queue.On("Stats").Return(stats)

		log := mocks.NewLogger(t)
		log.On("Error", mock.AnythingOfType("string"))

		// time.Hour guarantees that the scheduler will run only for 1 tick
		scheduler := schedule.NewScheduler(time.Hour, log, queue, semaphore.NewWeighted(2))
		assert.EqualValues(t, 0, scheduler.Stats().Panics())

		err := scheduler.Run(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.EqualValues(t, 1, scheduler.Stats().Panics())
	})

	t.Run("context is respected when semaphore is full", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		var (
			firstTaskCalled  bool
			secondTaskCalled bool
		)

		queue := mocks.NewQueue(t)
		queue.On("PopBatch", mock.AnythingOfType("time.Time")).
			Return([]func(context.Context){
				func(ctx context.Context) {
					firstTaskCalled = true
					cancel()
				},
				func(ctx context.Context) {
					secondTaskCalled = true
				},
			}).Once()

		stats := mocks.NewQueueStats(t)

		queue.On("Stats").Return(stats)

		scheduler := schedule.NewScheduler(time.Hour, mocks.NewLogger(t), queue, semaphore.NewWeighted(1))
		assert.EqualValues(t, 0, scheduler.Stats().Panics())

		err := scheduler.Run(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.True(t, firstTaskCalled, "first task must be called")
		assert.False(t, secondTaskCalled, "second task must not be called")
	})
}
