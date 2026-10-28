package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	httpapi "github.com/unsafemango/memory-wall/internal/http"
	"github.com/unsafemango/memory-wall/internal/store"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	siteURL := os.Getenv("SITE_URL")
	if siteURL == "" {
		siteURL = "http://localhost:3000"
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	server := httpapi.NewServer(httpapi.Deps{
		Events:      store.NewEventStore(db),
		Invitations: store.NewInvitationStore(db),
		Submissions: store.NewSubmissionStore(db),
		SiteURL:     siteURL,
	})

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", server.Routes()); err != nil {
		log.Fatal(err)
	}
}
