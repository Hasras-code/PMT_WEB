# Deployment and operations

## Local development

Requirements are Go 1.26.9 or newer, Node.js 22, npm, Docker Compose, and Python 3 for the setup and migration-name scripts.

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

## Initial platform administrator

The current schema already contains the global `PLATFORM_ADMIN` role, platform-scoped permissions, `user_platform_roles`, and append-only audit logging. No schema migration is required for initial provisioning. Do not insert a password-bearing account or add a public setup endpoint.

Use this sequence for the first administrator:

1. In the intended production database, confirm the database name and inspect existing platform administrators. The `000011_seed_platform_admin` migration contains a legacy conditional assignment for `as123123` / `test@example.com`, so confirm whether that exact legacy assignment exists before proceeding:

   ```sql
   SELECT current_database();

   SELECT u.id, u.email, u.status, u.email_verified_at,
          r.code AS role, upr.assigned_at,
          latest.new_values->>'source' AS assignment_source
   FROM user_platform_roles upr
   JOIN users u ON u.id = upr.user_id
   JOIN roles r ON r.id = upr.role_id AND r.scope = upr.scope
   LEFT JOIN LATERAL (
       SELECT a.new_values
       FROM audit_logs a
       WHERE a.entity_type = 'user'
         AND a.entity_id = u.id
         AND a.action = 'PLATFORM_ROLE_ASSIGNED'
         AND a.new_values->>'role' = r.code
       ORDER BY a.created_at DESC, a.id DESC
       LIMIT 1
   ) latest ON true
   WHERE r.code = 'PLATFORM_ADMIN'
   ORDER BY upr.assigned_at;
   ```

   If an administrator is already assigned, do not run the one-time command. Confirm the account is intended and usable, or use the existing admin UI to appoint a replacement before removing an obsolete administrator. Do not remove the legacy assignment based only on its email; confirm who controls the account first.

2. If there is no platform administrator, register the intended administrator through the normal frontend registration flow using an active cohort, then complete email verification. This creates the ordinary student account and preserves the existing hashed password and verification flow.

3. Deploy the backend revision containing the updated `cmd/admin` binary. The operational tools image is built separately with `make docker-tools-cloud-run`; do not put the tools binary into the public API image. No migration step is needed for this feature.

4. From a controlled operator environment with the production `DATABASE_URL` supplied from Secret Manager (do not print or paste the URL into logs), run:

   ```sh
   go run ./cmd/admin bootstrap-platform-admin \
     --database postgres \
     --confirm-database postgres \
     --email 'admin@example.edu' \
     --confirm-email 'admin@example.edu'
   ```

   Replace the email with the exact verified account. The command confirms `current_database()` is `postgres`, refuses any existing platform-admin assignment, uses a transaction-scoped lock to prevent concurrent attempts, and writes the role plus audit event in one transaction. It never changes a password. If your production database has another name, use that exact name in both database arguments. A mismatch or prior admin assignment is a hard failure.

5. Sign in through the existing LMS login with the account’s existing email and password. Confirm the platform admin area is visible and that `/v1/me/access` reports `PLATFORM_ADMIN`; then test one permitted platform operation. Platform permissions are loaded from PostgreSQL for each operation, not trusted from JWT role claims.

6. There are no platform-admin username/password environment variables to remove. Keep `AUTH_BASIC_USER` and `AUTH_BASIC_PASS`: they protect the separate health endpoints and are not LMS credentials.

7. If the wrong account was promoted, restore access through the account’s normal password-reset flow, sign in, assign the intended account `PLATFORM_ADMIN`, then remove the incorrect assignment using the admin UI. Keep at least one known-good administrator during the handoff. If the only admin account cannot be recovered through its verified email, stop and use a separately reviewed database recovery procedure with a backup and an audit record; do not roll back the authorization migration or reset its password through this bootstrap command.

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
