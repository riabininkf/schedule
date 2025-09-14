package schedule

// TaskQueueStats contains statistics about a task queue.
type TaskQueueStats struct {
	tasks             uint64
	estimatedTaskSize uint64
	usedBytes         uint64
	capacityBytes     uint64
}

// Tasks returns the number of tasks in the queue.
func (s *TaskQueueStats) Tasks() uint64 {
	return s.tasks
}

// EstimatedTaskSize returns the estimated size of a single task in the queue.
func (s *TaskQueueStats) EstimatedTaskSize() uint64 {
	return s.estimatedTaskSize
}

// UsedBytes returns the number of bytes that are currently used by the queue.
func (s *TaskQueueStats) UsedBytes() uint64 {
	return s.usedBytes
}

// UsedMiB returns the number of megabytes that are currently used by the queue.
func (s *TaskQueueStats) UsedMiB() float64 {
	return float64(s.usedBytes) / (1024 * 1024)
}

// CapacityBytes returns the total capacity of the queue in bytes.
func (s *TaskQueueStats) CapacityBytes() uint64 {
	return s.capacityBytes
}
