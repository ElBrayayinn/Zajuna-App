package workers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOrderedFanoutClaimsContiguousFIFOBeforeParallelWork(t *testing.T) {
	const n = 6
	const concurrency = 2

	var claimMu sync.Mutex
	claims := make([]int, 0, n)
	var inFlight int32
	var maxInFlight int32
	started := make([]int32, n)

	firstWindow := make(chan struct{})
	release := make(chan struct{})

	runErr := make(chan error, 1)
	go func() {
		runErr <- orderedFanout(context.Background(), n, concurrency, func(ctx context.Context, index int) error {
			claimMu.Lock()
			claims = append(claims, index)
			claimMu.Unlock()
			atomic.StoreInt32(&started[index], 1)

			current := atomic.AddInt32(&inFlight, 1)
			for {
				prev := atomic.LoadInt32(&maxInFlight)
				if current <= prev || atomic.CompareAndSwapInt32(&maxInFlight, prev, current) {
					break
				}
			}
			if current == int32(concurrency) {
				select {
				case firstWindow <- struct{}{}:
				default:
				}
			}
			select {
			case <-release:
			case <-ctx.Done():
				atomic.AddInt32(&inFlight, -1)
				return ctx.Err()
			}
			atomic.AddInt32(&inFlight, -1)
			return nil
		})
	}()

	select {
	case <-firstWindow:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first contiguous concurrency window")
	}

	for i := 0; i < concurrency; i++ {
		if atomic.LoadInt32(&started[i]) != 1 {
			t.Fatalf("expected index %d in the first window", i)
		}
	}
	for i := concurrency; i < n; i++ {
		if atomic.LoadInt32(&started[i]) != 0 {
			t.Fatalf("index %d must not start before earlier indices are claimed", i)
		}
	}

	for i := 0; i < n; i++ {
		release <- struct{}{}
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fanout did not finish")
	}

	claimMu.Lock()
	defer claimMu.Unlock()
	if len(claims) != n {
		t.Fatalf("expected %d claims, got %#v", n, claims)
	}
	for i, index := range claims {
		if index != i {
			t.Fatalf("claim order must be contiguous FIFO 0..n-1, got %#v", claims)
		}
	}
	if got := atomic.LoadInt32(&maxInFlight); got > int32(concurrency) {
		t.Fatalf("max in-flight %d exceeds concurrency %d", got, concurrency)
	}
}

func TestOrderedFanoutQueuesAllIndicesBeforeWorkersRun(t *testing.T) {
	const n = 8
	queued := make([]int, 0, n)
	// orderedFanout fills 0..n-1 before spawning work; mirror that contract here
	// and assert workers then claim without skipping.
	for i := 0; i < n; i++ {
		queued = append(queued, i)
	}
	var claims []int
	var mu sync.Mutex
	err := orderedFanout(context.Background(), n, 3, func(ctx context.Context, index int) error {
		mu.Lock()
		claims = append(claims, index)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, value := range queued {
		if value != i {
			t.Fatalf("pre-queue must be 0..n-1: %#v", queued)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for i, value := range claims {
		if value != i {
			t.Fatalf("claims must stay FIFO without skipping: %#v", claims)
		}
	}
}
