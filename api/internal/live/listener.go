package live

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Channel is the Postgres notification channel the triggers send on.
const Channel = "memory_wall_changes"

// Listen holds one database connection that LISTENs for changes and
// publishes them to the hub, until ctx is cancelled. If the connection
// drops it reconnects, then tells every page to reload in case it missed
// something in between.
func Listen(ctx context.Context, db *pgxpool.Pool, hub *Hub, logger *slog.Logger) {
	reconnecting := false
	for ctx.Err() == nil {
		err := listenOnce(ctx, db, hub, logger, func() {
			if reconnecting {
				hub.publishAll("RESYNC")
				logger.Info("change listener reconnected")
			}
		})
		if ctx.Err() != nil {
			return
		}
		logger.Error("change listener stopped, retrying", "error", err)
		reconnecting = true
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func listenOnce(ctx context.Context, db *pgxpool.Pool, hub *Hub, logger *slog.Logger, onReady func()) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN "+Channel); err != nil {
		return err
	}
	onReady()

	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var c Change
		if err := json.Unmarshal([]byte(n.Payload), &c); err != nil {
			logger.Warn("bad change notification", "payload", n.Payload, "error", err)
			continue
		}
		hub.Publish(c)
	}
}
