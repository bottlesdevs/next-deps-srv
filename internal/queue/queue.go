package queue

import (
	"context"
	"sync"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/bucket"
	"github.com/bottlesdevs/next-deps-srv/internal/email"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
	"github.com/google/uuid"
)

const maxConcurrent = 3

// logHub holds SSE subscriber channels for a job.
type logHub struct {
	mu   sync.Mutex
	subs []chan string
	done bool
	buf  []string
}

func (h *logHub) subscribe() chan string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 64)
	for _, line := range h.buf {
		ch <- line
	}
	if h.done {
		close(ch)
		return ch
	}
	h.subs = append(h.subs, ch)
	return ch
}

func (h *logHub) emit(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf = append(h.buf, line)
	for _, ch := range h.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (h *logHub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.done = true
	for _, ch := range h.subs {
		close(ch)
	}
	h.subs = nil
}

type BuildQueue struct {
	s       *store.Store
	backend bucket.Backend
	mailer  *email.Mailer
	sem     chan struct{}
	mu      sync.Mutex
	hubs    map[string]*logHub
}

func New(s *store.Store, backend bucket.Backend, mailer *email.Mailer) *BuildQueue {
	return &BuildQueue{
		s:       s,
		backend: backend,
		mailer:  mailer,
		sem:     make(chan struct{}, maxConcurrent),
		hubs:    make(map[string]*logHub),
	}
}

// Subscribe returns a channel that receives live log lines for a job.
// Channel is closed when the job finishes. Buffered replays prior lines.
func (q *BuildQueue) Subscribe(jobID string) <-chan string {
	q.mu.Lock()
	h, ok := q.hubs[jobID]
	q.mu.Unlock()
	if !ok {
		ch := make(chan string)
		close(ch)
		return ch
	}
	return h.subscribe()
}

func (q *BuildQueue) Submit(ctx context.Context, dep models.Dependency) (models.BuildJob, error) {
	job := models.BuildJob{
		ID:           uuid.NewString(),
		DependencyID: dep.ID,
		Status:       "queued",
		StartedAt:    time.Now(),
	}
	job, err := q.s.CreateJob(ctx, job)
	if err != nil {
		return job, err
	}

	h := &logHub{}
	q.mu.Lock()
	q.hubs[job.ID] = h
	q.mu.Unlock()

	go q.run(dep, job, h)
	return job, nil
}

func (q *BuildQueue) run(dep models.Dependency, job models.BuildJob, h *logHub) {
	q.sem <- struct{}{}
	defer func() { <-q.sem }()

	job.Status = "running"
	_ = q.s.UpdateJob(context.Background(), job)

	dep.Status = "building"
	_ = q.s.UpdateDep(context.Background(), dep)

	runJob(context.Background(), job, dep, q.s, q.backend, q.mailer, h)
	h.close()
}

// Queue is a simple string-job queue for testing and lightweight use.
type Queue struct {
	jobs chan string
	sem  chan struct{}
}

// NewTestQueue creates a Queue with the given max concurrency.
func NewTestQueue(maxWorkers int) *Queue {
	return &Queue{
		jobs: make(chan string, 256),
		sem:  make(chan struct{}, maxWorkers),
	}
}

// Push enqueues a job ID.
func (q *Queue) Push(jobID string) {
	q.jobs <- jobID
}

// StartWorker starts processing jobs using the provided worker function.
func (q *Queue) StartWorker(worker func(jobID string)) {
	go func() {
		for jobID := range q.jobs {
			q.sem <- struct{}{}
			go func(id string) {
				defer func() { <-q.sem }()
				worker(id)
			}(jobID)
		}
	}()
}
