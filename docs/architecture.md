# Architecture

PMT_WEB is one Git repository with two independently buildable applications:

- The repository root is a Go module that builds the HTTP API and operational commands.
- `frontend/` is a React, TypeScript, and Vite application.

The applications share an HTTP contract, but neither build consumes the other application's output. The intended production topology is Cloudflare Pages for the frontend, Google Cloud Run for the API, Supabase PostgreSQL through its shared Transaction Pooler, and Cloudflare R2 for object storage.

## Repository boundaries

```text
cmd/                 Go executable entry points
internal/            Private backend packages and feature services
migrations/          Ordered PostgreSQL schema migrations
frontend/            React/Vite application and frontend configuration
docs/                Architecture, deployment, security, and API documentation
scripts/             Reusable repository tooling
.github/workflows/   Independent backend and frontend checks
```

`go.mod` remains at the repository root. There is no Go workspace and no JavaScript workspace manager because the repository contains one Go module and one frontend application.

## Backend

`cmd/api` owns process startup and composes configuration, PostgreSQL, authentication, storage, HTTP handlers, and the Chi route tree. Its `openapi` command builds the same router without runtime services and emits the API contract.

Other executable entry points have concrete operational purposes:

- `cmd/migrate` applies the migrations in `migrations/`.
- `cmd/admin` performs audited bootstrap administration.
- `cmd/jobs` runs bounded notification and upload-cleanup work.

Feature packages remain under `internal/`. `internal/httpapi` translates HTTP requests and responses, while the feature packages own application operations and SQL. `internal/store` contains shared persistence operations. Platform-specific configuration, database, mail, and storage code lives under `internal/platform`.

The structure deliberately preserves the existing application and does not introduce ports/adapters, a second Go module, or generic utility packages.

## Frontend

The frontend lives entirely under `frontend/`. Pages and shared components use the central Axios client in `frontend/src/api/client.ts`. The public `VITE_API_URL` setting contains the API origin; endpoint calls already include the `/v1` prefix.

The frontend is suitable for Cloudflare Pages with:

- Root directory: `frontend`
- Build command: `npm run build`
- Output directory: `dist`

Backend secrets and privileged storage credentials must never be exposed as `VITE_` variables.

## Data and external services

PostgreSQL is the source of truth for identities, permissions, application data, and stored-object metadata. Migrations are sequential files at the repository root and are applied explicitly rather than during API startup.

Local development uses private filesystem storage under `data/files`. The current storage implementation validates content and issues short-lived application capabilities, but it is not suitable for horizontally scaled Cloud Run instances. Production Cloud Run deployment therefore requires an R2 adapter behind the storage boundary. R2 credentials remain in the backend, and the browser receives only API-authorized upload or download URLs.

Mail is delivered through SMTP; local development uses Mailpit. Production credentials belong in the backend runtime secret store.

## Deployment boundaries

```text
frontend/ ── Cloudflare Pages ── HTTPS ── Cloud Run API
                                             │
                                      ┌──────┴──────┐
                                      │             │
                              Supabase PostgreSQL   R2
```

The root Dockerfile's default target contains only the API. Its `tools` target contains migration, admin, and jobs commands for explicit operational use. The API image never depends on `frontend/dist`.

GitHub Actions validates backend and frontend changes independently. Cloudflare Pages may deploy through its Git integration. A Cloud Run deployment workflow should use GitHub OIDC with Google Workload Identity Federation rather than a stored service-account key.

## Security boundaries

The API remains authoritative for authentication and authorization. Frontend visibility checks improve navigation but do not grant access. PostgreSQL is never accessed by the browser. R2 and mail credentials are backend-only. See [security.md](security.md) for the role and permission model.
