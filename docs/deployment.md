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
- Future R2 access and secret keys
- Production mail credentials

Cloud Run may provide non-secret configuration directly:

- `APP_ENV=production`
- `PUBLIC_API_URL`
- `CORS_ALLOWED_ORIGINS`
- `COOKIE_SECURE=true`
- `DB_MAX_CONNS`
- `JWT_ISSUER`
- `JWT_AUDIENCE`
- Future R2 endpoint and bucket names

Cloudflare Pages receives only public build-time settings:

```text
VITE_API_URL=https://api.example.edu
```

The frontend currently includes `/v1` in each API path, so `VITE_API_URL` must be the origin without a trailing `/v1`.

## Backend image

Build the Cloud Run API image from the repository root:

```sh
docker build --target api -t university-lms-api .
```

The default final stage is also `api`, so `docker build -t university-lms-api .` is equivalent. The image contains the API binary and does not contain Node, frontend source, or frontend build output.

Operational commands use the separate tools target:

```sh
docker build --target tools -t university-lms-tools .
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

Store the complete runtime connection string in Google Secret Manager and expose it to the API as `DATABASE_URL`. The configured Supabase shared Transaction Pooler endpoint is:

```text
postgresql://postgres.bcwmnprcgemkgioiocum:PERCENT_ENCODED_PASSWORD@aws-0-ap-south-1.pooler.supabase.com:6543/postgres?sslmode=require
```

The password placeholder must be replaced inside Secret Manager, not committed. Percent-encode reserved URL characters in the password. The supplied connection details identify the database as `postgres`; use `/pmtdb` only if a database with that exact name has actually been created and verified.

Port 6543 is for API runtime traffic. The Go pool uses `pgx.QueryExecModeExec`, so it does not create named prepared statements that depend on a persistent PostgreSQL server session. Start with `DB_MAX_CONNS=5` and cap the Cloud Run service at three instances.

The API and `cmd/migrate` both read this same `DATABASE_URL` value.

The API reads Cloud Run's `PORT` automatically when `HTTP_ADDR` is unset. Production configuration requires HTTPS and secure cookies.

### Storage prerequisite

The current backend stores files on a local filesystem. Cloud Run's filesystem is ephemeral and cannot safely support multiple instances. Do not deploy file-upload workloads to Cloud Run until a tested Cloudflare R2 implementation replaces the local production backend. The local implementation remains appropriate for development.

## Cloudflare Pages

Connect the same Git repository to Cloudflare Pages with:

```text
Root directory: frontend
Build command: npm run build
Build output: dist
```

Set `VITE_API_URL` in Pages for each environment. Add the exact Pages origin to the API's `CORS_ALLOWED_ORIGINS`. Cloudflare Pages can deploy through its Git integration independently of backend releases.

## Migrations

Migrations remain under `migrations/` and are owned by the backend:

```sh
make migrate-up
make migrate-down
make migration name=description
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
docker build --target api -t university-lms-api .
docker build --target tools -t university-lms-tools .
```

Also verify migrations against a fresh PostgreSQL database and review `git status`, deleted files, environment files, generated output, and the lock file before committing.

## Troubleshooting

- Configuration failure: check the required database URL, a non-placeholder JWT secret, secure production URLs, CORS origins, and Basic-auth credentials.
- Database unavailable: inspect the Supabase pooler endpoint, TLS settings, credentials, and database readiness before restarting the API.
- Verification email missing: inspect Mailpit locally or the production mail provider logs.
- Refresh unexpectedly revoked: check whether concurrent requests reused the same rotating refresh token.
- Upload conflict: request a new immutable upload capability.
- Cross-cohort 404: confirm the entity belongs to the selected cohort and is visible in its current state.
