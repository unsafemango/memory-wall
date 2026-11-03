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
- **Live updates come from Postgres.** Triggers announce every new, approved, hidden, or deleted memory with `NOTIFY`. The API keeps one connection that `LISTEN`s and passes changes to open pages over Server-Sent Events. The manage page sees everything as it arrives, and the public page only hears about approved memories, so pending ones stay private. Because Postgres does the announcing, this also works with more than one API instance. If the connection drops, pages refetch when it comes back. Behind a proxy, make sure it doesn't buffer `text/event-stream` responses (the API sends `X-Accel-Buffering: no` for nginx).
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

Compose applies migrations automatically. To run them by hand, from `api/`:

```bash
DATABASE_URL="postgres://memorywall:devpassword@localhost:5432/memorywall?sslmode=disable" go run ./cmd/migrate up
```

The migration files are embedded in that command, and the same command runs on each deploy. `go run ./cmd/migrate version` shows where the database is.

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
| `TRUST_PROXY_HEADERS` | `true` behind a proxy you control, so rate limits use the real client IP (set in `fly.toml`) |

Web (`web/.env`):

| Variable              | What it's for                                                          |
| --------------------- | ---------------------------------------------------------------------- |
| `NEXT_PUBLIC_API_URL` | API address for browser requests. Inlined at build time                |
| `API_URL`             | API address for server-side requests (`http://api:8080` in Docker)     |

## Deploying

The setup is the web app on **Vercel**, the API on **Fly.io**, Postgres on **Neon**, and files on **Cloudflare R2**, under one domain you own:

- `app.yourdomain.com`: the web app (Vercel)
- `api.yourdomain.com`: the API (Fly.io)

Using two subdomains of the same domain matters. Browsers count them as the same site, so the login cookie works. Separate default domains (`*.vercel.app` and `*.fly.dev`) would break logins. Replace `yourdomain.com` below with your domain.

### 1. Database: Neon

1. Create a project in a region close to where the API will run.
2. Copy the **direct** connection string, the one whose host doesn't contain `-pooler`. Live updates use Postgres `LISTEN`, which doesn't work through Neon's connection pooler. Make sure it ends with `?sslmode=require`.

There's no need to run migrations yourself. The API's deploy does it.

### 2. Files: Cloudflare R2

1. Create a bucket named `memory-wall`.
2. In the bucket's **Settings → CORS policy**, paste [`deploy/r2-cors.json`](deploy/r2-cors.json) with your web app's address in `AllowedOrigins`.
3. Under **R2 → Manage API tokens**, create a token with **Object Read & Write** on that bucket. Note the access key ID, the secret access key, and the S3 endpoint, `https://<account-id>.r2.cloudflarestorage.com`.

### 3. API: Fly.io

Install [flyctl](https://fly.io/docs/flyctl/install/), run `fly auth login`, then from `api/`:

```bash
fly apps create memory-wall-api
```

If that name is taken, pick another and change `app` in [`api/fly.toml`](api/fly.toml) to match. Then set the secrets, filling in your values:

```bash
fly secrets set \
  DATABASE_URL="postgres://...neon.tech/neondb?sslmode=require" \
  SITE_URL="https://app.yourdomain.com" \
  S3_ENDPOINT="https://<account-id>.r2.cloudflarestorage.com" \
  S3_PUBLIC_ENDPOINT="https://<account-id>.r2.cloudflarestorage.com" \
  S3_ACCESS_KEY="..." \
  S3_SECRET_KEY="..." \
  S3_BUCKET="memory-wall" \
  INVITATION_LINK_KEY="$(openssl rand -hex 32)" \
  ANTHROPIC_API_KEY="..."
```

`INVITATION_LINK_KEY` must never change after people start using the app, or older personal links can no longer be shown. If you'll ever need to set up the API again, generate the key yourself first and keep a copy in a password manager.

Deploy, then give it your subdomain:

```bash
fly deploy
fly certs add api.yourdomain.com
```

`fly certs add` prints the DNS records to add at your domain's DNS provider. Each deploy runs `/bin/migrate up` before starting the new version (`release_command` in `fly.toml`). If a migration fails, the old version keeps running. Check it's up with `curl https://api.yourdomain.com/api/healthz`.

### 4. Web app: Vercel

1. Import the GitHub repo and set **Root Directory** to `web`. Vercel detects Next.js.
2. Add one environment variable, `NEXT_PUBLIC_API_URL` = `https://api.yourdomain.com`, then deploy.
3. In **Settings → Domains**, add `app.yourdomain.com` and create the DNS record Vercel shows.

From then on, Vercel redeploys the web app on every push to `main`.

### 5. Automatic API deploys

The CI workflow deploys the API to Fly after the tests pass on `main`. To turn it on, create a token with `fly tokens create deploy -a memory-wall-api` and add it to the GitHub repo as an Actions secret named `FLY_API_TOKEN` (**Settings → Secrets and variables → Actions**). Until then, that CI step just skips.

### 6. Check it works

Sign up at `https://app.yourdomain.com`, create an event, and open its share link in a private window. Then:

1. Leave a note and a photo as a guest.
2. Back on the manage page, check the memory shows up without a reload.
3. Approve it, and check it appears on the event page, also without a reload.
4. Upload a short video and check that its thumbnail appears a few seconds later.

### Good to know

- Fly may start two machines on the first deploy. That's fine: the job worker and live updates work across machines. Rate limits are counted per machine, so `fly scale count 1` keeps them exact.
- Back up the database with Neon's point-in-time restore, and check its retention on your plan.
- Logs: `fly logs` for the API, and the Vercel dashboard for the web app.

## Notes

- **MinIO image:** MinIO no longer publishes official Docker images, so this uses `bitnamilegacy/minio` for local development. It creates the `memory-wall` bucket on startup. This build has no web console, so use the `mc` CLI or any S3 client to look at buckets.
- **Changed Postgres credentials?** Postgres only reads them when the volume is first created. Reset with `docker compose down -v`.
