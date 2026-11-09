# Memory Wall

Memory Wall is a private, shared page for any occasion: a birthday, a wedding, a graduation, a farewell. Guests receive an invitation link and leave a note, photo, or video. The organizer reviews each submission and publishes the collection on a single page.

## Features

- **Events:** organizers create an event with its own page, date, and closing message.
- **Invitations:** a reusable share link for large groups, plus single-use personal links created individually or in bulk from a list of names.
- **Moderation:** submissions are held for review by default and can be approved, hidden, or deleted.
- **Photo and video uploads:** files upload directly to object storage, and videos are processed in the background.
- **Writing assistant:** guests can request three drafts in a chosen tone, optionally built around a personal detail.
- **Live updates:** new submissions reach the organizer's dashboard, and approved memories reach the event page, without a reload.

## Tech stack

| Area            | Technology                                                        |
| --------------- | ----------------------------------------------------------------- |
| API             | Go, Chi, pgx, `log/slog`                                          |
| Web             | Next.js (App Router), TypeScript, Tailwind CSS                    |
| Database        | PostgreSQL                                                        |
| Object storage  | S3-compatible: MinIO in development, Cloudflare R2 in production  |
| Background jobs | PostgreSQL job queue, ffmpeg                                      |
| AI              | Claude API, through the official Go SDK                           |
| Infrastructure  | Docker, GitHub Actions, Fly.io, Vercel, Neon                      |

## Architecture

```
Browser ──► Next.js ──► Go API ──► PostgreSQL
   │                      │  ▲          │
   │                      │  └─ LISTEN ─┘
   │                      ├──► Claude API
   │                      └──► Job worker (ffmpeg) ──► Object storage
   │                                                       ▲
   └────────────────── presigned PUT and GET ──────────────┘
```

### Design decisions

**Direct-to-storage uploads.** The API validates the file type and size, then issues a short-lived presigned URL for a key it generates. Files go straight from the browser to the bucket, so the API only handles small JSON requests. Before a submission is saved, the API confirms that the object exists, belongs to the event, and is within the size limit. The bucket is private, and media is served through signed URLs that expire after an hour.

**PostgreSQL job queue.** Background work runs from a database table rather than a separate message broker, which keeps one stateful system to operate at this scale. Workers claim jobs with `FOR UPDATE SKIP LOCKED`, and failed jobs retry with backoff. Video jobs are enqueued in the same transaction as their submission.

**Live updates through `LISTEN`/`NOTIFY`.** Database triggers announce each change to a memory. The API holds one listening connection and forwards changes to open pages over Server-Sent Events. Because the database is the source of these events, this works across multiple API instances. The public stream only reports changes to approved memories, so pending submissions stay private.

**Invitation security.** Session and personal invitation tokens are 32 random bytes, and only their SHA-256 hashes are used for lookup. Personal links are also stored encrypted with AES-256-GCM so organizers can retrieve them. Single use is enforced in the same transaction that saves a submission, which prevents races. Share links are stored in plain form by design, since they are meant to be distributed, and they can be disabled or rotated.

**Authorization and request protection.** Passwords are hashed with Argon2id. Ownership checks are part of the SQL for every organizer query, and resources belonging to another organizer return `404`. Session cookies are `HttpOnly` and `SameSite=Lax`, and state-changing requests from other origins are rejected. Authentication and guest endpoints are rate limited per client.

## Project structure

```
memory-wall/
├── api/
│   ├── cmd/api/            API server entry point
│   ├── cmd/migrate/        Database migration command
│   ├── internal/ai/        Writing assistant
│   ├── internal/auth/      Passwords, tokens, encryption
│   ├── internal/http/      Routes, handlers, middleware
│   ├── internal/jobs/      Job queue, worker, video processing
│   ├── internal/live/      Change notifications
│   ├── internal/storage/   Object storage client
│   ├── internal/store/     Database queries
│   ├── migrations/         SQL migrations
│   └── scripts/seed.sql    Development seed data
├── web/
│   ├── app/                Pages (event, invitation, dashboard, auth)
│   └── lib/                API client, hooks, shared types
├── deploy/                 Storage CORS policy
└── docker-compose.yml      Local development stack
```

## Getting started

### Prerequisites

- Docker
- Go 1.26 and Node.js 22, to run the API or web app outside Docker

### Run locally

```bash
cp api/.env.example api/.env
cp web/.env.example web/.env
docker compose up -d --build
```

Compose starts PostgreSQL and MinIO, applies migrations, then starts the API and web app.

| Service        | Address                           |
| -------------- | --------------------------------- |
| Web            | http://localhost:3000             |
| API            | http://localhost:8080/api/healthz |
| MinIO (S3 API) | http://localhost:9000             |
| PostgreSQL     | localhost:5432                    |

Development credentials are defined in `docker-compose.yml`. To load sample data:

```bash
docker exec -i postgres psql -U memorywall -d memorywall < api/scripts/seed.sql
```

### Migrations

Migrations are embedded in the `migrate` command. Compose applies them automatically, and they can also be run by hand from `api/`:

```bash
DATABASE_URL="postgres://memorywall:devpassword@localhost:5432/memorywall?sslmode=disable" go run ./cmd/migrate up
```

### Tests

From `api/`:

```bash
DATABASE_URL="postgres://memorywall:devpassword@localhost:5432/memorywall?sslmode=disable" go test ./...
```

Tests that need a database are skipped when `DATABASE_URL` is not set.

## Configuration

### API

| Variable                                                   | Description                                                                 |
| ---------------------------------------------------------- | --------------------------------------------------------------------------- |
| `DATABASE_URL`                                             | PostgreSQL connection string. Must be a direct connection, since live updates use `LISTEN` |
| `SITE_URL`                                                 | Web app origin, used for CORS, origin checks, and invitation links          |
| `S3_ENDPOINT`                                              | Object storage endpoint used by the API                                     |
| `S3_PUBLIC_ENDPOINT`                                       | Object storage endpoint used by browsers                                    |
| `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` | Bucket settings and credentials                                             |
| `INVITATION_LINK_KEY`                                      | 64 hex characters that encrypt personal invitation links. Must stay the same once in use |
| `ANTHROPIC_API_KEY`                                        | Enables the writing assistant (optional)                                    |
| `TRUST_PROXY_HEADERS`                                      | Set to `true` behind a trusted proxy, so rate limits use the client IP      |

### Web

| Variable              | Description                                                                              |
| --------------------- | ---------------------------------------------------------------------------------------- |
| `API_URL`             | API address for server-side requests. Also the target of the `/api/*` forwarding         |
| `NEXT_PUBLIC_API_URL` | API address for browser requests. When unset, the browser uses the site's own `/api` path |

## Deployment

In production, the web app runs on Vercel, the API on Fly.io, the database on Neon, and object storage on Cloudflare R2. The API configuration is in [`api/fly.toml`](api/fly.toml), and the storage CORS policy is in [`deploy/r2-cors.json`](deploy/r2-cors.json).

GitHub Actions runs the tests, lint, type checks, and builds on every push. After a successful run on `main`, the API deploys to Fly.io, where migrations run as a release step before the new version starts. Vercel deploys the web app from `main`.
