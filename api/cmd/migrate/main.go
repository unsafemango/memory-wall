// Command migrate applies the database migrations embedded in the binary.
// It ships in the API image so deploys can run it before the new API
// starts, e.g. as Fly's release_command.
//
//	migrate up       apply every new migration (the default)
//	migrate version  print the current version
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/unsafemango/memory-wall/migrations"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fail("DATABASE_URL is not set")
	}
	// The pgx v5 driver registers itself as pgx5://.
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(dbURL, prefix) {
			dbURL = "pgx5://" + strings.TrimPrefix(dbURL, prefix)
		}
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		fail("read migrations", "error", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		fail("connect", "error", err)
	}
	defer m.Close()

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "up":
		err := m.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database is up to date")
			return
		}
		if err != nil {
			fail("migrate up", "error", err)
		}
		v, _, _ := m.Version()
		slog.Info("migrated", "version", v)
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			fail("read version", "error", err)
		}
		fmt.Printf("version %d (dirty: %v)\n", v, dirty)
	default:
		fail("unknown command, use up or version", "command", cmd)
	}
}

func fail(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
