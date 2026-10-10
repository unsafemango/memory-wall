# Memory Wall

A private memory wall for any event, a birthday, a wedding, a graduation, a going-away party. People get an invite link, leave a note, photo, or video, and the event's organizer gets everything collected on one page. Includes an AI-assisted writing helper for people who don't know what to say.

## Stack

- **API:** Go, Chi router, Postgres
- **Web:** Next.js, TypeScript
- **Storage:** S3-compatible (MinIO locally, Cloudflare R2 in production), presigned direct uploads
- **Auth:** Invitation tokens for contributors, Argon2id-hashed password for admin/moderation

## Status

Early scaffolding. Following a phased build plan, nothing beyond basic project structure is built yet.

## Running locally
