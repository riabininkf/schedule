# Schedule

A lightweight, memory-aware, panic-safe task scheduler for Go that executes functions at specified times with controlled concurrency and observable metrics.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/riabininkf/schedule"
)

type FavouriteLogger struct{}

func (l *FavouriteLogger) Error(msg string) {}

func main() {
	scheduler := schedule.NewScheduler(
		time.Second, // check tasks every second
		&FavouriteLogger{},
		schedule.NewTaskQueue( // default queue, provided by this package
			10*1024*1024, // 1 MiB queue size
		),
		semaphore.NewWeighted(5), // 5 workers will serve the queue
	)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Go(func() {
		// the only possible error is context.Canceled.
		_ = scheduler.Run(ctx)
	})

	if err := scheduler.Add(time.Now().Add(time.Second), func(ctx context.Context) {
		fmt.Println("1 second later")
	}); err != nil {
		panic(err)
	}

	if err := scheduler.Add(time.Now().Add(time.Second*5), func(ctx context.Context) {
		fmt.Println("5 seconds later")
		cancel()
	}); err != nil {
		panic(err)
	}

	wg.Wait()
}

```

## Why not spawn goroutines with time.Sleep or use time.AfterFunc?
Using a centralized scheduler has several practical advantages over spawning goroutine timers:

- Fewer goroutines, lower memory overhead
    - time.Sleep blocks a goroutine until wake-up; time.AfterFunc creates and manages timers per task.
    - With many delayed tasks, creating a goroutine/timer per task scales poorly (heap growth, GC pressure).
    - A single scheduler goroutine wakes on a periodic tick and dispatches due tasks in batches.

- Rate limiting
    - A semaphore limits concurrent task execution globally.
    - With per-task goroutines, concurrency explodes uncontrollably unless you add extra plumbing everywhere.

- Deterministic ordering and batching
    - A priority queue ensures tasks with the earliest execution time run first.
    - Batched pops lower lock contention and reduce context switches.
