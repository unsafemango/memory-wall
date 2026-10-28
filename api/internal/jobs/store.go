package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// maxAttempts is how many times a job runs before it's marked failed.
const maxAttempts = 3

type Job struct {
	ID       string
	Type     string
	Payload  json.RawMessage
	Attempts int
}

// Store is a job queue in a Postgres table. It's enough at this scale and
// keeps everything in one database.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Enqueue adds a job to run as soon as a worker is free.
func (s *Store) Enqueue(ctx context.Context, jobType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `INSERT INTO jobs (type, payload) VALUES ($1, $2)`, jobType, body)
	return err
}

// claim takes the oldest due job and marks it running. FOR UPDATE SKIP LOCKED
// means two workers never pick up the same job. Returns nil when there's
// nothing to do.
func (s *Store) claim(ctx context.Context) (*Job, error) {
	var j Job
	err := s.db.QueryRow(ctx,
		`UPDATE jobs SET status = 'running', attempts = attempts + 1
		 WHERE id = (
		     SELECT id FROM jobs
		     WHERE status = 'pending' AND run_at <= now()
		     ORDER BY run_at
		     FOR UPDATE SKIP LOCKED
		     LIMIT 1
		 )
		 RETURNING id, type, payload, attempts`,
	).Scan(&j.ID, &j.Type, &j.Payload, &j.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (s *Store) markDone(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `UPDATE jobs SET status = 'done', last_error = NULL WHERE id = $1`, id)
	return err
}

// markFailed retries the job later with a growing delay, or gives up after
// maxAttempts.
func (s *Store) markFailed(ctx context.Context, j *Job, jobErr error) error {
	if j.Attempts >= maxAttempts {
		_, err := s.db.Exec(ctx, `UPDATE jobs SET status = 'failed', last_error = $2 WHERE id = $1`, j.ID, jobErr.Error())
		return err
	}
	delay := time.Duration(j.Attempts*j.Attempts) * 30 * time.Second
	_, err := s.db.Exec(ctx,
		`UPDATE jobs SET status = 'pending', last_error = $2, run_at = now() + $3::interval WHERE id = $1`,
		j.ID, jobErr.Error(), delay.String(),
	)
	return err
}

// resetStuck puts jobs left running by a crashed or restarted worker back in
// the queue.
func (s *Store) resetStuck(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `UPDATE jobs SET status = 'pending' WHERE status = 'running'`)
	return err
}
