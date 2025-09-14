package schedule

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/riabininkf/gox/container"
)

// ErrCapacityExceeded is returned when the queue capacity is exceeded.
var ErrCapacityExceeded = fmt.Errorf("tasks capacity exceeded")

// NewTaskQueue creates a new *TaskQueue instance.
func NewTaskQueue(
	capacityBytes uint64,
) *TaskQueue {
	taskSizeBytes := uint64(unsafe.Sizeof(task{})) // approximately 32 Bytes

	return &TaskQueue{
		capacityBytes: capacityBytes,
		taskSizeBytes: taskSizeBytes,
		stats: &TaskQueueStats{
			estimatedTaskSize: taskSizeBytes,
			capacityBytes:     capacityBytes,
		},
		heap: container.NewHeap[task](func(a, b task) bool {
			return a.executeAt.Before(b.executeAt)
		}),
	}
}

type (
	// TaskQueue is a priority queue of tasks. Tasks with the closest execution time are at the top of the queue.
	TaskQueue struct {
		capacityBytes uint64

		taskSizeBytes uint64
		stats         *TaskQueueStats
		mux           sync.RWMutex
		heap          *container.Heap[task]
	}

	task struct {
		executeAt time.Time
		fn        func(ctx context.Context)
	}
)

// Add adds a new task to the queue with the given execution time.
// Tasks with the closest execution time are at the top of the queue.
// If the queue capacity is exceeded, ErrCapacityExceeded is returned.
func (q *TaskQueue) Add(execAt time.Time, fn func(ctx context.Context)) error {
	q.mux.Lock()
	defer q.mux.Unlock()

	if q.stats.usedBytes+q.taskSizeBytes > q.capacityBytes {
		return ErrCapacityExceeded
	}

	q.heap.Push(task{executeAt: execAt, fn: fn})

	q.stats.usedBytes += q.taskSizeBytes
	q.stats.tasks++

	return nil
}

// PopBatch extracts the next batch of tasks from the queue that should be executed by the given time.
func (q *TaskQueue) PopBatch(execAt time.Time) []func(ctx context.Context) {
	q.mux.Lock()
	defer q.mux.Unlock()

	result := make([]func(ctx context.Context), 0)

	for {
		if t, ok := q.heap.Top(); !ok || t.executeAt.After(execAt) {
			return result
		}

		if t, ok := q.heap.Pop(); ok {
			result = append(result, t.fn)

			q.stats.usedBytes -= q.taskSizeBytes
			q.stats.tasks--
		}
	}
}

// Stats return a snapshot of the queue stats.
func (q *TaskQueue) Stats() QueueStats {
	q.mux.RLock()
	defer q.mux.RUnlock()

	stats := *q.stats

	return &stats
}
