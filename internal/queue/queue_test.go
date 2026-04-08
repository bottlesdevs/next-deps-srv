package queue_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/queue"
)

func TestQueueSubmitAndProcess(t *testing.T) {
	bq := queue.NewTestQueue(3)

	var processed int32
	bq.StartWorker(func(jobID string) {
		atomic.AddInt32(&processed, 1)
	})

	bq.Push("job-1")
	bq.Push("job-2")
	bq.Push("job-3")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&processed) == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if n := atomic.LoadInt32(&processed); n != 3 {
		t.Errorf("expected 3 processed, got %d", n)
	}
}

func TestQueueConcurrency(t *testing.T) {
	bq := queue.NewTestQueue(2)

	var active int32
	var maxActive int32

	bq.StartWorker(func(_ string) {
		n := atomic.AddInt32(&active, 1)
		for {
			cur := atomic.LoadInt32(&maxActive)
			if n <= cur || atomic.CompareAndSwapInt32(&maxActive, cur, n) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&active, -1)
	})

	for i := 0; i < 6; i++ {
		bq.Push("job")
	}

	time.Sleep(500 * time.Millisecond)
	if m := atomic.LoadInt32(&maxActive); m > 2 {
		t.Errorf("concurrency exceeded limit: max active was %d, limit is 2", m)
	}
}
