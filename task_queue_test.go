package schedule_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/riabininkf/schedule"
)

const taskSizeBytes uint64 = 32

func TestTaskQueue(t *testing.T) {
	testTask := func(ctx context.Context) {}

	t.Run("capacity exceeded", func(t *testing.T) {
		capacityBytes := taskSizeBytes
		const MiB = 1 << 20

		taskQueue := schedule.NewTaskQueue(capacityBytes)

		assertStats(t, taskQueue.Stats(), 0, capacityBytes)

		assert.NoError(t, taskQueue.Add(time.Now(), testTask), "first task must be added")
		assert.ErrorIs(t, taskQueue.Add(time.Now(), testTask), schedule.ErrCapacityExceeded,
			"second task must not be added")

		assertStats(t, taskQueue.Stats(), 1, capacityBytes)
	})

	// ... existing code ...

	t.Run("positive case", func(t *testing.T) {
		capacityBytes := taskSizeBytes * 2

		taskQueue := schedule.NewTaskQueue(capacityBytes)
		assertStats(t, taskQueue.Stats(), 0, capacityBytes)

		assert.NoError(t, taskQueue.Add(time.Now(), testTask), "first task must be added")
		assert.NoError(t, taskQueue.Add(time.Now().Add(time.Hour), testTask), "second task must be added")

		assertStats(t, taskQueue.Stats(), 2, capacityBytes)

		assert.Len(t, taskQueue.PopBatch(time.Now()), 1, "only one task must be popped")
		assertStats(t, taskQueue.Stats(), 1, capacityBytes)
	})
}

// ... existing code ...

func assertStats(t *testing.T, stats schedule.QueueStats, tasks uint64, capacityBytes uint64) {
	t.Helper()

	const MiB = 1 << 20 // 1,048,576

	taskSize := stats.EstimatedTaskSize()
	expectedUsedBytes := tasks * taskSize
	expectedUsedMiB := float64(expectedUsedBytes) / float64(MiB)

	expectedCapacityBytes := capacityBytes

	assert.EqualValues(t, tasks, stats.Tasks())
	assert.EqualValues(t, taskSize, stats.EstimatedTaskSize())
	assert.EqualValues(t, expectedUsedBytes, stats.UsedBytes())
	assert.InDelta(t, expectedUsedMiB, stats.UsedMiB(), 1e-8)

	assert.EqualValues(t, expectedCapacityBytes, stats.CapacityBytes())
}
