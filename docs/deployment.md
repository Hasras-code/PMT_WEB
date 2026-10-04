# Deployment and operations

## Local development

Requirements are Go 1.26.8 or newer, Node.js 22, npm, Docker Compose, and Python 3 for the setup and migration-name scripts.

From the repository root:

```sh
make init
make db-up
make migrate-up
make run
```

In another terminal:

```sh
cd frontend
npm ci
npm run dev
```

The API listens on `http://localhost:8080`, the frontend on `http://localhost:5173`, and Mailpit on `http://localhost:8025`. Local files are written to `data/files`. The storage implementation creates this directory when needed.

To run the API and migrations in containers:

```sh
make init
docker compose --profile app up --build -d
```

Compose is a local-development environment. It is not a Cloud Run emulator.

## Environment ownership

Keep actual environment files out of Git. `.env.example` documents backend/local settings and `frontend/.env.example` documents public browser settings.

Google Cloud Secret Manager should provide sensitive backend values such as:

- `DATABASE_URL`
- `JWT_SECRET`
- `AUTH_BASIC_PASS`
- `R2_ACCESS_KEY_ID`
- `R2_SECRET_ACCESS_KEY`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`

Cloud Run may provide non-secret configuration directly:

- `APP_ENV=production`
- `PUBLIC_API_URL`
- `FRONTEND_URL`
- `CORS_ALLOWED_ORIGINS`
- `COOKIE_SECURE=true`
- `DB_MAX_CONNS`
- `JWT_ISSUER`
- `JWT_AUDIENCE`
- `STORAGE_PROVIDER=r2`
- `R2_ENDPOINT`
- `R2_REGION=auto`
- `R2_PRIVATE_BUCKET`
- `R2_PUBLIC_BUCKET`
- `R2_PUBLIC_BASE_URL`
- `UPLOAD_URL_TTL_SECONDS`
- `DOWNLOAD_URL_TTL_SECONDS`
- `SMTP_ADDR`
- `SMTP_TLS_MODE=starttls` or `implicit`
- `MAIL_FROM`
- `AUTH_BASIC_USER`

Cloudflare Pages receives only public build-time settings:

```text
VITE_API_URL=https://api.example.edu
```

The frontend currently includes `/v1` in each API path, so `VITE_API_URL` must be the origin without a trailing `/v1`.

## Backend image

Build the Cloud Run API image for Cloud Run's required Linux amd64 platform from the repository root:

```sh
make docker-cloud-run
# Or, when publishing directly:
docker buildx build --platform linux/amd64 --target api \
  -t REGION-docker.pkg.dev/PROJECT/REPOSITORY/university-lms-api:GIT_SHA --push .
```

The default final stage is also `api`. The image contains the API binary and does not contain Node, frontend source, or frontend build output. Always verify that the published manifest contains `linux/amd64`; a plain local build on an Apple Silicon host produces an arm64 image that Cloud Run cannot start.

Operational commands use the separate tools target:

```sh
make docker-tools-cloud-run
```

## Cloud Run and Supabase PostgreSQL

The recommended backend release flow is:

1. Backend CI passes on a pull request.
2. A merge to `main` builds the API target.
3. The image is pushed to Artifact Registry.
4. Reviewed migrations run as a separate Cloud Run Job or controlled CI step.
5. Cloud Run deploys the immutable image revision.
6. Readiness and a representative authenticated request are verified.

Use GitHub OIDC and Google Workload Identity Federation for deployment. Do not store a service-account JSON key in the repository or GitHub secrets.

Store separate runtime and migration connection strings in Google Secret Manager. Both are exposed to their own workload with the existing variable name `DATABASE_URL`; the API never receives the migration secret.

The database name in the URL path is configurable. Point both the API and migration job at the same database that contains the application schema; use `/pmtdb` if that is the database you migrated. The sample below uses `/postgres` from the supplied Supabase connection details.

The API uses the shared Transaction Pooler:

```text
postgresql://postgres.bcwmnprcgemkgioiocum:PERCENT_ENCODED_PASSWORD@aws-0-ap-south-1.pooler.supabase.com:6543/postgres?sslmode=require
```

The password placeholder must be replaced inside Secret Manager, not committed. Percent-encode reserved URL characters in the password. The supplied connection details identify the database as `postgres`; use `/pmtdb` only if a database with that exact name has actually been created and verified.

Port 6543 is for API runtime traffic. The Go pool uses `pgx.QueryExecModeExec`, so it does not create named prepared statements that depend on a persistent PostgreSQL server session. Start with `DB_MAX_CONNS=5` and cap the Cloud Run service at three instances.

The controlled migration job must use the shared Session Pooler because `golang-migrate` holds a session advisory lock:

```text
postgresql://postgres.bcwmnprcgemkgioiocum:PERCENT_ENCODED_PASSWORD@aws-0-ap-south-1.pooler.supabase.com:5432/postgres?sslmode=require
```

`cmd/migrate` rejects the transaction-pooler endpoint to prevent an unsafe production migration. Apply migration `000025_object_storage` before deploying the R2-enabled API and require migration version 25 with `dirty=false` before shifting traffic.

The API reads Cloud Run's `PORT` automatically when `HTTP_ADDR` is unset. Production configuration requires HTTPS and secure cookies.

### Cloudflare R2 storage

Set `STORAGE_PROVIDER=r2` for Cloud Run. Production startup rejects local storage and performs a bounded availability check against both configured buckets. Browser uploads use short-lived presigned PUT URLs and private downloads use short-lived presigned GET URLs, so file bytes do not pass through Cloud Run. Configure the bucket CORS policy and a production custom public media domain described in [Object storage](storage.md).

Before enabling R2 for an environment with existing local files, apply migration `000025_object_storage`, run `make migrate-storage-dry-run`, and then run `make migrate-storage`. The migration utility verifies each object before marking its database metadata as R2 and does not delete local copies.

## Cloudflare Pages

Connect the same Git repository to Cloudflare Pages with:

```text
Root directory: frontend
Build command: npm run build
Build output: dist
```

Set `VITE_API_URL` in Pages for each environment. Add the exact Pages origin to the API's `CORS_ALLOWED_ORIGINS`. Cloudflare Pages can deploy through its Git integration independently of backend releases.

## Production mail

Registration verification and password reset use authenticated SMTP. Mailtrap Email Sending works with the existing SMTP adapter: configure `SMTP_ADDR=live.smtp.mailtrap.io:587`, `SMTP_TLS_MODE=starttls`, `SMTP_USERNAME=api`, and `SMTP_PASSWORD` with the Mailtrap SMTP token from the verified sending domain's Transactional Stream integration. Set `MAIL_FROM` to a sender address on that verified domain. Keep the token in Secret Manager.

Set `FRONTEND_URL` to the exact browser app origin and include that same origin in `CORS_ALLOWED_ORIGINS`. Verification messages contain a link to `/verify#token=…`; the page fills the one-time token and asks the user to confirm. The token is in the URL fragment, so the browser does not send it to the web server, and the page removes it from the address bar before confirmation. Production startup rejects HTTP frontend URLs, missing SMTP credentials, plaintext SMTP, and localhost mail servers.

## Initial Cloud Run service settings

Use region `asia-south1`, public ingress, and allow unauthenticated infrastructure access; application JWT and permission middleware protect private routes. Start with 1 vCPU, 512 MiB, concurrency 20, request timeout 120 seconds, minimum instances 0, and maximum instances 3. Use a dedicated service account with access only to the required Secret Manager secrets. Leave `HTTP_ADDR` unset so Cloud Run's injected `PORT` is authoritative.

Use the default TCP startup probe initially. `/health/live` and `/health/ready` remain Basic-authenticated operational endpoints; `/health/ready` checks PostgreSQL with a two-second deadline.

## Migrations

Migrations remain under `migrations/` and are owned by the backend:

```sh
make migrate-up
make migrate-down
make migration name=description
make migrate-storage-dry-run
make migrate-storage
```

Never rewrite or rename an applied migration. Do not apply migrations during normal API startup. Review destructive down migrations before use; production rollback usually requires restoring compatible data rather than automatically migrating down.

## Background work

`make jobs` runs one bounded pass of notification fan-out and expired-upload/session cleanup. In production, run the jobs binary from a scheduler or Cloud Run Job with the same database and storage configuration as the API. Repeating a completed pass is safe.

## Backups and rollback

Back up PostgreSQL and object storage as one logical dataset. Record the application revision and migration version with each backup.

For an application-only rollback, route Cloud Run traffic to the previous healthy immutable revision. Confirm that the previous binary is compatible with all migrations already applied. For a data-affecting migration, follow its reviewed recovery plan or restore into a separate environment first.

Before a frontend rollback, select the previous successful Cloudflare Pages deployment. Frontend and backend rollback independently as long as their API contract remains compatible.

## Verification

Before release, run:

```sh
make test
make test-race
make audit
make build
make frontend-lint
make frontend-typecheck
make frontend-build
make docs
git diff --exit-code docs/openapi.json
make docker-cloud-run
make docker-tools-cloud-run
```

Also verify migrations against a fresh PostgreSQL database and review `git status`, deleted files, environment files, generated output, and the lock file before committing.

## Troubleshooting

- Configuration failure: check that `DATABASE_URL` is a TLS-required PostgreSQL URL with a database name matching the migrated schema, a non-placeholder JWT secret, secure production URLs, CORS origins, Basic-auth credentials, authenticated TLS SMTP, and R2 settings.
- Database unavailable: inspect the Supabase pooler endpoint, TLS settings, credentials, and database readiness before restarting the API.
- Verification email missing: inspect Mailpit locally or the production mail provider logs.
- Refresh unexpectedly revoked: check whether concurrent requests reused the same rotating refresh token.
- Upload conflict: request a new immutable upload capability.
- Cross-cohort 404: confirm the entity belongs to the selected cohort and is visible in its current state.
