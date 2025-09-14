package schedule

// Stats is a collection of queue stats.
type Stats struct {
	QueueStats
	panics uint64
}

// Panics returns the number of panics in scheduled tasks.
func (s *Stats) Panics() uint64 {
	return s.panics
}
