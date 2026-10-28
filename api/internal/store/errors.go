package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrDuplicate = errors.New("already exists")
var ErrNotFound = errors.New("not found")

// ErrInvitationUsed means the invitation was already used, or expired, by
// the time the submission was saved.
var ErrInvitationUsed = errors.New("invitation already used")

// mapDuplicate turns a unique-constraint violation into ErrDuplicate.
func mapDuplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}
