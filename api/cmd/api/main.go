package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unsafemango/memory-wall/internal/ai"
	httpapi "github.com/unsafemango/memory-wall/internal/http"
	"github.com/unsafemango/memory-wall/internal/jobs"
	"github.com/unsafemango/memory-wall/internal/storage"
	"github.com/unsafemango/memory-wall/internal/store"
)

func main() {
	// ctx is cancelled on Ctrl-C or when Docker stops the container.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	var aiClient *ai.Client
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		aiClient = ai.NewClient(key)
	} else {
		log.Println("ANTHROPIC_API_KEY is not set, the writing helper is turned off")
	}

	files := storage.New(storage.Config{
		Endpoint:       os.Getenv("S3_ENDPOINT"),
		PublicEndpoint: os.Getenv("S3_PUBLIC_ENDPOINT"),
		Region:         os.Getenv("S3_REGION"),
		AccessKey:      os.Getenv("S3_ACCESS_KEY"),
		SecretKey:      os.Getenv("S3_SECRET_KEY"),
		Bucket:         os.Getenv("S3_BUCKET"),
	})

	server := httpapi.NewServer(httpapi.Deps{
		Organizers:  store.NewOrganizerStore(db),
		Sessions:    store.NewSessionStore(db),
		Events:      store.NewEventStore(db),
		Invitations: store.NewInvitationStore(db),
		Submissions: store.NewSubmissionStore(db),
		AI:          aiClient,
		Storage:     files,
		SiteURL:     siteURL,
		TrustProxy:  os.Getenv("TRUST_PROXY_HEADERS") == "true",
	})

	// The job worker runs alongside the HTTP server for the life of the program.
	worker := jobs.NewWorker(jobs.NewStore(db), slog.Default())
	workerDone := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(workerDone)
	}()

	srv := &http.Server{Addr: ":8080", Handler: server.Routes()}
	go func() {
		log.Println("listening on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	<-workerDone
}
