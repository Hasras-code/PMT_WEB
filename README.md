# PMT university LMS API

A Go REST API for multiple university cohorts and a single public platform gallery. PostgreSQL runs locally in Docker Compose. PDFs and images are stored on local disk, outside the database and outside any static web directory. There is no frontend, achievement model, fund management, album model, or per-batch gallery.

## Run locally

Requirements: Go 1.26.8 or newer, Docker with Compose, and Python 3 for the small setup/migration-name scripts. Go's automatic toolchain download supports older installed Go versions when network access is available.

```sh
make init
make db-up
make migrate-up
make run
```

`make init` creates a gitignored `.env` with a random signing secret. It never overwrites existing configuration. If configuring manually, copy `.env.example` and replace the placeholder secret with at least 32 random bytes, for example using `openssl rand -base64 48`.

- API: http://localhost:8080
- Liveness: http://localhost:8080/health/live
- Readiness: http://localhost:8080/health/ready
- Mailpit: http://localhost:8025 (verification and password-reset emails)
- PostgreSQL: localhost:5432, database/user `lms`, local-only password `lms_local_only`
- Files when using `make run`: `./data/files`

Health endpoints require HTTP Basic Auth. Configure `AUTH_BASIC_USER` and `AUTH_BASIC_PASS` in `.env`, then provide those credentials when opening either health URL. These credentials are separate from LMS user accounts and platform-admin permissions.

The Compose ports bind to loopback. The database and file contents persist in named Docker volumes when using the containerized API. `docker compose down` preserves volumes; adding `-v` destroys them.

To run the API itself in Docker:

```sh
make init
docker compose --profile app up --build -d
```

The migration container must complete successfully before the API starts. Containers run the application as a non-root user. Host mode and container mode use different local file directories; keep one mode for a dataset, or migrate the files along with the database.

## Structure and dependencies

`cmd/api` composes the HTTP server, middleware boundaries, route tree, PostgreSQL pool, authentication, mail, and local storage. Its `openapi` command mode generates the API contract from the same router used by the running server. `cmd/migrate` applies golang-migrate files using its pgx/v5 adapter. `cmd/admin` provides audited bootstrap commands. `cmd/jobs` processes notifications and expired uploads.

`internal/httpapi` contains thin Chi/net/http handlers, strict JSON decoding, response mapping and middleware. Domain packages contain explicit application operations and parameterized PostgreSQL queries. `authorization` centralizes current database permission checks. `platform` supplies typed configuration, bounded pgxpool connections, SMTP and filesystem storage. Domain operations do not depend on Chi, HTTP statuses or external storage-provider SDK types. Shared interfaces are limited to actual boundaries such as mail delivery, storage and query access.

Stack: Go, Chi v5, PostgreSQL 17, pgx/v5, golang-migrate, jwt/v5, bcrypt and slog. Dependencies are pinned in `go.mod`/`go.sum`.

## Authentication

1. `POST /v1/auth/register` with `student_number`, `combination`, `first_name`, `last_name`, `display_name`, `email`, `password`. Combination is required and accepts `PMT-ICT` or `PMT-CS` (case-insensitive input is stored canonically).
2. Read the email in Mailpit. Submit its token to `POST /v1/auth/verify-email` as `{"token":"..."}`.
3. Log in through `POST /v1/auth/login` with email/password. Verification does not log you in automatically.
4. Send `Authorization: Bearer ACCESS_TOKEN` to authenticated endpoints.
5. Rotate through `POST /v1/auth/refresh` using `{"refresh_token":"..."}` before the 15-minute access token expires.

Passwords require at least 12 Unicode characters and at most 72 UTF-8 bytes; they are not trimmed. Email uniqueness is case-insensitive. Registration, resend verification and forgot-password return generic accepted responses. No API returns verification/reset tokens. A mail failure leaves the pending account intact.

Verification tokens expire after 24 hours; password-reset tokens after 30 minutes. Only SHA-256 hashes of random 32-byte tokens are stored. Passwords use bcrypt with cost 12. Reset revokes all sessions.

Refresh tokens are opaque, rotate on every use, and have an absolute 30-day session expiry. Spent hashes are retained to detect replay. Reuse revokes the entire login session. Concurrent refresh attempts can therefore invalidate the session: consumers must serialize refresh requests, and an ambiguously lost rotation response may require login again.

JWTs contain identity and session ID, never authoritative roles. Every authenticated request checks current account and session state. Logout, logout-all, session deletion, suspension and password reset therefore invalidate subsequent access requests immediately.

For browser-compatible cookie mode, login with `"transport":"cookie"` and an allowed `Origin`. The refresh token is returned only in a host-only HttpOnly SameSite=Lax cookie scoped to `/v1/auth`. Cookie refresh/logout require an allowed Origin. Do not mix cookie and JSON refresh credentials. `COOKIE_SECURE=true` is required in production. Cross-site SameSite=None mode is intentionally not enabled.

## Cohorts, roles and bootstrap

Registration creates a global account without cohort membership. Approval creates an ACTIVE membership and its STUDENT role in one transaction. A deferred PostgreSQL constraint additionally prevents an active membership from committing without STUDENT. Lifecycle statuses, rather than deleting the base role, represent suspension or graduation.

First register, verify and log in as the intended operator. Obtain their UUID with `GET /v1/me`. Export `.env` when invoking Go commands directly (Make does this automatically):

```sh
set -a
. ./.env
set +a

go run ./cmd/admin create-batch --actor USER_UUID --batch cohort-2026 --name 'Cohort 2026' --year 2026
go run ./cmd/admin create-membership --actor USER_UUID --batch cohort-2026 --student ST12345
go run ./cmd/admin assign-first-batch-rep --actor USER_UUID --batch cohort-2026 --student ST12345
go run ./cmd/admin assign-platform-admin --actor USER_UUID --student ST12345
```

CLI access and database credentials are privileged operator capabilities. `--actor` supplies an existing active account for auditing; it is not an authentication replacement for untrusted CLI users. Do not expose this CLI as a public service.

Representatives can approve memberships with `POST /v1/batches/{batchID}/members` and body `{"user_id":"..."}`. Promotion uses `POST /v1/batches/{batchID}/members/{membershipID}/roles` and body `{"role":"BATCH_REP"}`. Removal uses DELETE on the corresponding role-code path. Additional responsibilities never replace STUDENT. Normal API operations protect the last active representative from role removal/membership suspension; operator access provides recovery.

See [security and permissions](docs/security.md) for the permission matrix. Public committee titles do not grant access. Platform administrators have global gallery and platform-management permissions but no automatic access to private cohort content.

## PDF upload, versions and downloads

1. Request `POST /v1/batches/{batchID}/resources/uploads` with `file_name`, `mime_type: "application/pdf"`, and exact `size_bytes`.
2. PUT the raw file bytes to the returned `upload_url`, with the returned Content-Type. It is a signed five-minute capability and needs no Authorization header.
3. Create the resource through `POST /v1/batches/{batchID}/resources` with `upload_id`, `module_id`, `type`, and `title` (plus optional metadata).
4. Publish with `POST /v1/batches/{batchID}/resources/{resourceID}/publish`.
5. Authorize a download with `GET /v1/batches/{batchID}/resources/{resourceID}/download`; follow the returned five-minute URL.

For another version, authorize at `.../{resourceID}/versions/uploads`, then finalize at `.../{resourceID}/versions` with `upload_id` and optional `change_note`, size/compression/page metadata. Updating a published resource's file requires `resource.publish` as well as `resource.update`.

Files are streamed to a private temporary file with a strict size limit, validated, synced, then atomically linked to a server-generated immutable key. Reusing an upload capability cannot replace a finalized file. Finalization binds owner, batch and purpose and consumes the intent transactionally. Cross-cohort modules are rejected by composite foreign keys. Reusing a consumed intent fails without creating another resource/version.

PDFs are limited to 50 MiB. Images allow JPEG, PNG and WebP up to 10 MiB; gallery thumbnails are limited to 1 MiB at finalization. Image signatures/dimensions and PDF signatures are checked; headers alone are not trusted. This is format validation, not malware scanning. PDF compression and OCR are not performed.

In this local-storage implementation, file bytes necessarily pass through Go. The storage interface allows replacement with object storage later. Do not deploy this filesystem backend onto an ephemeral or independently scaled container filesystem. An issued private download URL remains valid until expiry, even if membership changes in the meantime; new download authorizations reflect the change immediately.

Announcement attachments follow the same upload/confirmation pattern under `.../announcements/{announcementID}/attachments`. Profile image uploads use `/me/profile/image/uploads` followed by PATCH `/me/profile` with `upload_id`. Batch hero uploads use `.../profile/uploads` followed by PATCH `.../profile`. Event images use `.../events/{eventID}/uploads` followed by POST `.../events/{eventID}/image`.

## Global gallery and public portfolio

Only PLATFORM_ADMIN's `gallery.manage` grants gallery administration. Authorize display and thumbnail uploads through `POST /v1/admin/gallery/uploads`, upload both files, then POST `/v1/admin/gallery` with `display_upload_id`, `thumbnail_upload_id`, `alt_text`, and optional title/caption/taken_at. Publish explicitly at `.../{imageID}/publish`.

`GET /v1/public/gallery?limit=12&month=2026-09` returns published images and delivery URLs. Default limit is 12, maximum 20. Month filtering uses UTC half-open timestamp ranges on `COALESCE(taken_at,published_at)`. Results sort by that timestamp and UUID descending. Pass `next_cursor` as `cursor` and preserve the month filter. Cursors are authenticated. Ordering timestamps cannot be edited after creation/publication.

Public image routes recheck publication on every request and send no-store headers. Draft and archived gallery files have no public route. Gallery rows have no batch ownership. Public batch profiles, published PUBLIC events and public committee positions are available under `/v1/public/batches/{slug}`.

## Complaints and feedback

Anonymous submissions store `submitted_by = NULL`. Application logs exclude submission-request correlations and bodies; audits do not secretly store the author. Anonymous complaints are not returned in `/complaints/mine` and have no submitter conversation continuity in V1. Identified owners may view/reply to their own complaints. COMPLAINT_MANAGER can read/respond/resolve within its cohort. BATCH_REP alone cannot read private complaints.

Transitions are OPEN → IN_REVIEW → RESOLVED → CLOSED, with RESOLVED → IN_REVIEW reopening allowed. The database enforces anonymity checks and same-batch parent relationships. If using a proxy or external log collector, disable identity-bearing correlation for anonymous submissions there too; this service cannot erase third-party network logs.

## API contract and errors

See [OpenAPI](docs/openapi.json). Regenerate it with `make docs`, or open the development-only Swagger UI at `http://localhost:8080/swagger/`. The service has no application HTML pages. Dates use ISO 8601; timestamps use RFC 3339. Collection limits default to 20 and maximum 100, with offset capped at 100000, except the gallery cursor API described above.

PATCH fields generally preserve omitted values. Nullable fields currently treat JSON null as omitted; dedicated lifecycle operations control security-sensitive state. State-changing actions and logout are typically 204; creations return 201; authentication responses return 200 and enumeration-safe email requests return 202.

Errors have a stable envelope:

```json
{"error":{"code":"forbidden","message":"You are not allowed to perform this action","request_id":"..."}}
```

Malformed/unknown-field JSON is rejected. Validation uses 422, missing objects 404, insufficient permission 403, invalid state/duplicate constraints 409, and throttling 429. Database internals and credentials are never serialized.

## Configuration

| Variable | Default / purpose |
|---|---|
| `APP_ENV` | `development`; production requires HTTPS and secure cookies |
| `HTTP_ADDR` | `:8080`, or `PORT` when HTTP_ADDR is absent |
| `PUBLIC_API_URL` | `http://localhost:8080`; origin used for generated delivery URLs |
| `DATABASE_URL` | Required PostgreSQL DSN |
| `DB_MAX_CONNS` | 5 per process; account for replica count |
| `JWT_SECRET` | Required random secret, minimum 32 bytes; changing it invalidates all JWTs and file capabilities |
| `JWT_ISSUER` / `JWT_AUDIENCE` | `pmt-api` / `pmt-clients` |
| `STORAGE_DIR` | `./data/files`; private, persistent directory |
| `SMTP_ADDR` / `MAIL_FROM` | `localhost:1025` / `lms@localhost`; local Mailpit SMTP adapter |
| `CORS_ALLOWED_ORIGINS` | Comma-separated exact origins; no wildcard |
| `COOKIE_SECURE` | false for local HTTP; true for HTTPS |
| `TRUSTED_PROXY_CIDRS` | Empty: ignore forwarding headers. Configure only controlled proxies. |

SMTP is configured for local Mailpit. A production deployment should provide a TLS/authenticated mail adapter and least-privilege database credentials. This revision intentionally delivers the requested local infrastructure.

## Tests and operations

```sh
make test
make test-integration
TEST_DATABASE_URL='postgres://lms:lms_local_only@localhost:5432/lms?sslmode=disable' make test-race
make audit
make build
```

Integration tests use fresh, randomly named schemas and remove only those test schemas. The PostgreSQL test account needs CREATE SCHEMA privileges. Without TEST_DATABASE_URL, integration tests explicitly skip. CI supplies PostgreSQL and runs them, plus race detection, formatting, module verification, vet, Staticcheck, vulnerability checks, OpenAPI drift verification and a container build.

`make jobs` runs bounded notification fan-out, expired unconsumed upload cleanup, incomplete temporary-file cleanup, and expired session/token retention cleanup. Schedule it externally when deploying; notifications are not dependent on an in-process background loop. Repeating it is safe.

Migrations have sequential up/down pairs. `make migrate-down` reverses one migration and may destroy its data; use only for development or a reviewed recovery procedure. `make migration name=description` creates the next pair. System roles are installed by migrations; no sample credentials or production correctness depend on development seed data.

See [operations](docs/operations.md) for backup, deployment and recovery details.
