package main

import (
	"context"
	"errors"
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

	// One JSON object per log line, so logs can be searched by field.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fatal("DATABASE_URL is not set")
	}

	siteURL := os.Getenv("SITE_URL")
	if siteURL == "" {
		siteURL = "http://localhost:3000"
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fatal("connect to database", "error", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		fatal("ping database", "error", err)
	}

	var aiClient *ai.Client
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		aiClient = ai.NewClient(key)
	} else {
		logger.Warn("ANTHROPIC_API_KEY is not set, the writing helper is turned off")
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
		Logger:      logger,
	})

	// The job worker runs alongside the HTTP server for the life of the program.
	worker := jobs.NewWorker(jobs.NewStore(db), logger)
	worker.Handle(store.JobProcessVideo, jobs.NewVideoProcessor(files, store.NewMediaStore(db)).Handle)
	workerDone := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(workerDone)
	}()

	srv := &http.Server{Addr: ":8080", Handler: server.Routes()}
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fatal("http server", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "error", err)
	}
	<-workerDone
}

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
