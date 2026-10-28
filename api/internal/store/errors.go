package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrDuplicate = errors.New("already exists")
var ErrNotFound = errors.New("not found")

// mapDuplicate turns a unique-constraint violation into ErrDuplicate.
func mapDuplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}
