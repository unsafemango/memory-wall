# Memory Wall

A private memory wall for any event, a birthday, a wedding, a graduation, a going-away party. People get an invite link, leave a note, photo, or video, and the event's organizer gets everything collected on one page. Includes an AI-assisted writing helper for people who don't know what to say.

## Stack

- **API:** Go, Chi router, Postgres
- **Web:** Next.js, TypeScript
- **Storage:** S3-compatible (MinIO locally, Cloudflare R2 in production), presigned direct uploads
- **Auth:** Invitation tokens for contributors, Argon2id-hashed password for admin/moderation

## Status

Phase 0 (scaffolding) is done. The full local stack runs in Docker, and the Next.js app can reach the Go API. Phase 1 (data layer) has started with the initial schema migration. Following a phased build plan.

## Project structure

```
memory-wall/
  api/                Go API (Chi)
    cmd/api/          entry point
    internal/         http handlers, store, jobs
    migrations/       SQL migrations (golang-migrate)
  web/                Next.js app (App Router, TypeScript)
  docker-compose.yml  Postgres, MinIO, API, web
```

## Running locally

Requires Docker.

1. Create the env files from the examples:

   ```bash
   cp api/.env.example api/.env
   cp web/.env.example web/.env
   ```

2. Start everything:

   ```bash
   docker compose up -d --build
   ```

3. Check it's working:

   | Service       | URL                                  |
   | ------------- | ------------------------------------ |
   | Web           | http://localhost:3000                |
   | API health    | http://localhost:8080/api/healthz    |
   | MinIO (S3 API) | http://localhost:9000 (minioadmin / minioadmin) |
   | Postgres      | localhost:5432 (memorywall / devpassword) |

   The web page should show "API status: ok".

### Migrations

Migrations aren't run automatically yet. With [golang-migrate](https://github.com/golang-migrate/migrate) installed (`brew install golang-migrate`), run from `/api`:

```bash
migrate -database "postgres://memorywall:devpassword@localhost:5432/memorywall?sslmode=disable" -path migrations up
```

Use `localhost` here, not `postgres`. The `postgres` hostname in `api/.env` only resolves inside the Docker network.

### Notes

- **MinIO image:** MinIO no longer publishes official Docker images, so this uses `bitnamilegacy/minio` for local development. Production uses Cloudflare R2. This MinIO build has no web console, so use the `mc` CLI or any S3 client to look at buckets.
- **CORS:** the API only allows requests from `http://localhost:3000` right now.
- **Changed Postgres credentials?** Postgres only reads them when the volume is first created. Reset with `docker compose down -v`.
