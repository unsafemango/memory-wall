package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Handler runs one job. A returned error means the job is retried later.
type Handler func(ctx context.Context, payload []byte) error

// Worker polls the jobs table and runs each job with its type's handler.
type Worker struct {
	store    *Store
	handlers map[string]Handler
	logger   *slog.Logger
}

func NewWorker(store *Store, logger *slog.Logger) *Worker {
	return &Worker{store: store, handlers: map[string]Handler{}, logger: logger}
}

func (w *Worker) Handle(jobType string, h Handler) {
	w.handlers[jobType] = h
}

// Run processes jobs until ctx is cancelled. It's started as a goroutine
// next to the HTTP server.
func (w *Worker) Run(ctx context.Context) {
	if err := w.store.resetStuck(ctx); err != nil {
		w.logger.Error("reset stuck jobs", "error", err)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		// Drain everything that's due before waiting for the next tick.
		for w.processNextJob(ctx) {
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// processNextJob runs one job and reports whether there was one to run.
func (w *Worker) processNextJob(ctx context.Context) bool {
	job, err := w.store.claim(ctx)
	if err != nil {
		if ctx.Err() == nil {
			w.logger.Error("claim job", "error", err)
		}
		return false
	}
	if job == nil {
		return false
	}

	log := w.logger.With("job_id", job.ID, "job_type", job.Type, "attempt", job.Attempts)
	start := time.Now()

	err = w.run(ctx, job)
	if err != nil {
		log.Error("job failed", "error", err, "duration_ms", time.Since(start).Milliseconds())
		if err := w.store.markFailed(ctx, job, err); err != nil {
			log.Error("mark job failed", "error", err)
		}
		return true
	}

	log.Info("job done", "duration_ms", time.Since(start).Milliseconds())
	if err := w.store.markDone(ctx, job.ID); err != nil {
		log.Error("mark job done", "error", err)
	}
	return true
}

func (w *Worker) run(ctx context.Context, job *Job) (err error) {
	h, ok := w.handlers[job.Type]
	if !ok {
		return fmt.Errorf("no handler for job type %q", job.Type)
	}
	// A panicking job shouldn't take the whole API down with it.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job panicked: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return h(ctx, job.Payload)
}
