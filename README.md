# Memory Wall

A private memory wall for any event, a birthday, a wedding, a graduation, a going-away party. People get an invite link, leave a note, photo, or video, and the event's organizer gets everything collected on one page. Includes an AI-assisted writing helper for people who don't know what to say.

## What it does

- **Organizers** sign up and create an event with its own page (`/sarah-30th`). They invite guests with one share link for everyone (a group chat, an email, a flyer), or with personal links made one at a time or from a pasted list of names.
- **Guests** open their link, write a note or attach a photo or video, and can ask for help writing (three drafts in a tone they pick, optionally built around a detail they type).
- **Moderation** is on by default: submissions wait in the organizer's dashboard until they're approved, rejected, or deleted.
- **The event page** shows every approved memory as note cards and polaroids, followed by the organizer's closing message.

## Stack

- **API:** Go, Chi router, Postgres (pgx, no ORM), `log/slog` JSON logs
- **Web:** Next.js (App Router), TypeScript, Tailwind
- **Storage:** S3-compatible (MinIO locally, Cloudflare R2 in production), presigned direct uploads
- **Auth:** Organizer accounts (email + Argon2id-hashed password, Postgres-backed sessions) for creating events and moderating; a reusable share link per event plus single-use personal invitation links for contributors
- **AI:** Claude (Opus 5.5) through the official Go SDK, called only from the server
- **Background jobs:** a Postgres table and a worker goroutine; videos get a thumbnail, duration, and dimensions from ffmpeg

## Architecture

```
Browser ──► Next.js (web) ──► Go API ──► Postgres
   │                            │
   │      presigned PUT/GET     ├──► S3 bucket (MinIO / R2)
   └──────────────────────────► │
                                ├──► Claude API (writing helper)
                                └──► job worker (ffmpeg) ──► S3 bucket
```

The event page is a Server Component, so it fetches memories on the server. Everything else calls the API from the browser.

### Decisions worth knowing

- **Uploads go straight to the bucket.** The API checks the file type and size, then signs a 5-minute PUT URL for a key it generates (`events/<event-id>/<uuid>.jpg`). The browser uploads to the bucket directly, so the API only handles small JSON requests. Before saving the submission, the API checks that the object exists, belongs to the invitation's event, and is within the size limit.
- **The bucket is private.** Photos and videos come back as signed GET links that expire after an hour.
- **The job queue is a Postgres table.** At this scale it's simpler than Kafka or Redis and keeps one database to run. Workers claim jobs with `FOR UPDATE SKIP LOCKED`, failed jobs retry with a growing delay up to three times, and jobs left running by a restart are picked up again. A video's job is queued in the same transaction that saves it.
- **Two kinds of invite link.** Each event has a **share link** anyone can use as many times as they like. It's the easy way to invite a big group, and moderation keeps strangers off the page if it spreads. The organizer can turn it off, or reset it so the old link stops working. **Personal links** greet a guest by name, work once, and show who has posted. Organizers can make up to 200 at a time from a list of names and download them as a CSV.
- **Tokens are stored as hashes.** Session cookies and personal invitation links are 32 random bytes. Only their SHA-256 hash goes in the database, so a leaked backup can't be used to log in or submit. Passwords use Argon2id instead, because unlike random tokens they're guessable. Share tokens are the exception: they're stored as-is so the organizer can copy the link again, which is acceptable for a link that's meant to be passed around and can be reset. Personal links are also kept encrypted (AES-256-GCM, keyed by `INVITATION_LINK_KEY`) next to their hash, so the organizer can click a name to see its link again. A leaked database alone doesn't reveal them.
- **Personal links are single-use.** Saving a submission marks the invitation used in the same transaction, so two requests racing on one link can't both succeed. Expired, used, or turned-off links return `410 Gone` with a clear message.
- **Ownership checks live in the SQL.** Every organizer query joins on `events.owner_id`, and someone else's event or submission returns `404`, not `403`, so IDs can't be probed.
- **CSRF.** The session cookie is `HttpOnly` and `SameSite=Lax`, CORS only allows the web app's origin, and state-changing requests with a different `Origin` header are rejected.
- **Rate limits** are in-memory token buckets: per IP on login and signup, and per IP and per link-and-IP on guest endpoints (so a hundred guests on one share link each get their own allowance), with a tighter limit on the paid AI endpoint. This works for a single API instance. Running several would need a shared store (Redis, or a Postgres table).
- **Errors** always come back as `{"error": {"code": "...", "message": "..."}}`.

## Project structure

```
memory-wall/
  api/
    cmd/api/            entry point: config, wiring, HTTP server, job worker
    internal/http/      routes, handlers, middleware (auth, CSRF, rate limits, logging)
    internal/store/     Postgres queries, one file per table
    internal/auth/      Argon2id passwords, random tokens
    internal/storage/   S3 client: presigned URLs, uploads, deletes
    internal/jobs/      job queue, worker, video processing
    internal/ai/        Claude writing helper
    migrations/         SQL migrations (golang-migrate)
    scripts/seed.sql    dev seed data
  web/
    app/[slug]/         public event page
    app/invite/[token]/ guest page for leaving a memory
    app/dashboard/      organizer dashboard and moderation
    app/login, signup/  organizer auth
    lib/                API helpers, upload helper, shared types
  docker-compose.yml    Postgres, MinIO, migrations, API, web
  .github/workflows/    CI
```

## Running locally

Requires Docker.

1. Create the env files from the examples:

   ```bash
   cp api/.env.example api/.env
   cp web/.env.example web/.env
   ```

   To try the writing helper, put an Anthropic API key in `ANTHROPIC_API_KEY` in `api/.env`. Without one, everything else works and the helper says it isn't set up.

2. Start everything:

   ```bash
   docker compose up -d --build
   ```

   Compose starts Postgres, runs any new migrations, then starts the API and the web app.

3. Open http://localhost:3000, sign up, and create an event. On the event's Manage page, use **Invite someone** to make a link, then open it in a private window to leave a memory as a guest.

   | Service        | URL                                             |
   | -------------- | ----------------------------------------------- |
   | Web            | http://localhost:3000                           |
   | API health     | http://localhost:8080/api/healthz               |
   | MinIO (S3 API) | http://localhost:9000 (minioadmin / minioadmin) |
   | Postgres       | localhost:5432 (memorywall / devpassword)       |

### Seed data

`api/scripts/seed.sql` adds an event at `/sarah` (moderation off) and one invitation at http://localhost:3000/invite/dev-token-plaintext:

```bash
docker exec -i postgres psql -U memorywall -d memorywall < api/scripts/seed.sql
```

Personal invitations are single-use, so delete the invitation row and run it again for a fresh one. Every event also has a reusable share link, shown on its Manage page.

### Running the API outside Docker

Postgres and MinIO can stay in Docker:

```bash
docker compose up -d postgres minio migrate
```

Then, in `api/`, set `DATABASE_URL` and `S3_ENDPOINT` to use `localhost` instead of the `postgres` and `minio` hostnames, and run `go run ./cmd/api`. Those hostnames only resolve inside the Docker network. The video job needs `ffmpeg` installed (`brew install ffmpeg`).

### Migrations

Compose applies migrations automatically. To run them by hand, with [golang-migrate](https://github.com/golang-migrate/migrate) installed (`brew install golang-migrate`), from `api/`:

```bash
migrate -database "postgres://memorywall:devpassword@localhost:5432/memorywall?sslmode=disable" -path migrations up
```

New schema changes always go in a new numbered migration. Don't edit one that has already run.

### Tests

From `api/`:

```bash
DATABASE_URL="postgres://memorywall:devpassword@localhost:5432/memorywall?sslmode=disable" go test ./...
```

Database tests skip when `DATABASE_URL` isn't set. CI runs everything against a Postgres service, plus lint, a type check, and a build for the web app.

## Configuration

API (`api/.env`):

| Variable              | What it's for                                                       |
| --------------------- | ------------------------------------------------------------------- |
| `DATABASE_URL`        | Postgres connection string                                          |
| `SITE_URL`            | Web app origin. CORS, CSRF checks, and invitation links use it      |
| `S3_ENDPOINT`         | How the API reaches the bucket                                      |
| `S3_PUBLIC_ENDPOINT`  | How browsers reach the bucket. Presigned URLs are signed for it     |
| `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` | Bucket settings and credentials |
| `ANTHROPIC_API_KEY`   | Turns on the writing helper                                         |
| `INVITATION_LINK_KEY` | 64 hex characters (`openssl rand -hex 32`). Encrypts personal invite links so organizers can copy them again. Keep it secret and don't change it, or older links can't be shown |
| `TRUST_PROXY_HEADERS` | `true` behind a proxy you control, so rate limits use the real client IP |

Web (`web/.env`):

| Variable              | What it's for                                                          |
| --------------------- | ---------------------------------------------------------------------- |
| `NEXT_PUBLIC_API_URL` | API address for browser requests. Inlined at build time                |
| `API_URL`             | API address for server-side requests (`http://api:8080` in Docker)     |

## Deploying

A good setup is managed Postgres (Neon, Supabase, or Railway), the API on Fly.io or Railway, the web app on Vercel, and Cloudflare R2 for storage.

- **R2 CORS:** allow the web app's origin to `PUT` and `GET`:

  ```json
  [{ "AllowedOrigins": ["https://your-app.example"], "AllowedMethods": ["PUT", "GET"], "AllowedHeaders": ["*"] }]
  ```

- **Session cookie across domains:** browsers won't send the `SameSite=Lax` cookie from `your-app.vercel.app` to `your-api.fly.dev`. Set `API_URL` to the API's address and `NEXT_PUBLIC_API_URL` to an empty string when building the web app, and Next.js proxies `/api/*` to the API so the browser stays on one origin. Then set `SITE_URL` on the API to the web app's URL and `TRUST_PROXY_HEADERS=true`.
- **Migrations** should run as part of each deploy, before the new API version starts.
- Set `ANTHROPIC_API_KEY` as a secret on the API host.

## Notes

- **MinIO image:** MinIO no longer publishes official Docker images, so this uses `bitnamilegacy/minio` for local development. It creates the `memory-wall` bucket on startup. This build has no web console, so use the `mc` CLI or any S3 client to look at buckets.
- **Changed Postgres credentials?** Postgres only reads them when the volume is first created. Reset with `docker compose down -v`.
