# Operations

## Lifecycle

Start PostgreSQL and Mailpit with `docker compose up -d postgres mailpit`. Apply migrations explicitly through `make migrate-up`. Start the host API with `make run`, or run the entire application with `docker compose --profile app up --build -d`.

The API validates configuration once, logs structured JSON to stdout, uses a five-connection pool by default, bounds PostgreSQL statements to ten seconds, and gives shutdown fifteen seconds on SIGINT/SIGTERM. Liveness checks the process; readiness performs a two-second database ping. File streams have HTTP read/write timeouts and size limits.

Local files require persistent storage. The container uses `/data/files` in the `files_data` volume; the host process defaults to `data/files`. Do not run replicas with separate filesystem roots against one database. Runtime identity/authorization never resides exclusively in process memory.

## Background work

Run `make jobs` periodically from your own scheduler, or execute `/app/jobs` in an application container with the same environment and file volume. One run dispatches at most 50 notification events and processes up to 100 expired upload intents. Temporary partial files older than 24 hours are removed in bounded batches. It can be run again until the backlog is empty. There is no external message broker.

Notification delivery is database persistence only; external push delivery is deferred. Notification insertions deduplicate on event and recipient. `FOR UPDATE SKIP LOCKED` allows concurrent job execution without double fan-out. A rollback retains unprocessed work.

Uploads expire after fifteen minutes; their signed PUT capability lasts five minutes. Jobs leave an additional hour before deleting unconsumed files. No remote calls occur inside content transactions. Interrupted uploads leave only private data until cleanup; finalization and metadata commit atomically within PostgreSQL.

## Backups

Back up PostgreSQL and the file directory together. Database metadata references immutable local files. For a consistent local backup, pause API writes, run `pg_dump` against the lms database, copy the file volume, record the application/migration versions, then resume writes. Keep signing secrets in a separate secret store, not in source or database dumps.

Restore into a separate environment first, restore the matching files, apply only the intended migrations, and verify readiness plus a private PDF download and public gallery image before switching traffic. Avoid `docker compose down -v` unless destroying local data intentionally.

## Troubleshooting

- Configuration failure: check `.env`, a non-placeholder JWT_SECRET, DATABASE_URL, URL origins and file directory permissions.
- Database unavailable: `docker compose ps`, `docker compose logs postgres`, then retry readiness.
- Verification email missing: inspect Mailpit and structured mail-delivery warnings, then use resend verification. Accounts are not deleted on delivery failure.
- Refresh unexpectedly revoked: check whether two requests used the same refresh token. Reuse revokes the login session by design.
- Upload conflict: immutable keys cannot be overwritten; authorize a fresh upload. A consumed intent cannot create another entity.
- Cross-batch 404: the entity is not in the authorized route cohort, or its publication state is not visible.
- Gallery image unavailable: ensure the image is PUBLISHED and the files were preserved with the database.

No service automatically applies migrations during normal API startup. The Compose migration service is a separate deployment step. CI exercises fresh schemas; a production rollback needs a data-aware recovery procedure, not indiscriminate down migrations.
