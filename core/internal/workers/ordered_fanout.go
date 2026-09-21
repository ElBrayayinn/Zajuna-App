package workers

import (
	"context"
	"sync"
)

// orderedFanout runs n tasks with at most concurrency workers. Indices are
// placed on a FIFO queue as 0..n-1 before any worker starts. Workers claim the
// next index under a mutex so claim order is strictly contiguous (no skipping),
// then execute the work outside the mutex so up to concurrency tasks run in
// parallel Chromium sessions.
func orderedFanout(ctx context.Context, n, concurrency int, run func(ctx context.Context, index int) error) error {
	if n <= 0 {
		return nil
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > n {
		concurrency = n
	}

	queue := make(chan int, n)
	for index := 0; index < n; index++ {
		queue <- index
	}
	close(queue)

	errCh := make(chan error, concurrency)
	var claimMu sync.Mutex
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if err := ctx.Err(); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				claimMu.Lock()
				index, ok := <-queue
				claimMu.Unlock()
				if !ok {
					return
				}
				if err := run(ctx, index); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}()
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
