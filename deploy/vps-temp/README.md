# Temp VPS deploy (IP-only, no domain) — isolated bundle

## Connection details (recovered from previous sessions)

- PMT temp VPS IP: `95.211.43.93` (public IPv4, no scheme, no port)
- SSH: `ssh root@95.211.43.93`, repo path `/opt/pmt-web` (clone then run below)
- PMT ports (this bundle only): `80` (frontend), `8090` (API direct), `8025`
  (Mailpit temp — delete the allow rule after testing). Postgres stays
  `127.0.0.1:5432`.
- Occupied on this box — do NOT touch: `localhost:8080` (caddy),
  `8000/8501/6379/443` (trading).
- Separate box `95.211.126.202` (hermes trading agent, key `~/.ssh/hermes_vps`)
  is out of scope for this bundle — leave it running untouched.

All temp-deploy files live here so they are easy to recognize and delete before
final production. Nothing in this folder is imported by local dev.

## What you get

- `docker-compose.vps.yml` — overlay on top of root `docker-compose.yml`.
  Opens `80` (frontend+proxy), `${API_PORT:-8090}` (API direct, required because
  upload/download URLs are absolute `PUBLIC_API_URL/v1/files/...`), and
  `8025` (Mailpit temp). Postgres stays `127.0.0.1:5432` on the VPS.
  Port 8090 (not 8080) because the shared box already runs caddy on
  localhost:8080 and trading on 8000/8501/6379/443.
- `frontend.Dockerfile` — Vite build (`VITE_API_URL=http://<IP>:8090`) + nginx.
- `nginx.conf` — serves `dist/`, proxies `/v1/`, `/health/`, `/swagger/` to `api:8080`.
- `.env.vps.example` — template. Real `.env.vps` is gitignored.
- `deploy.sh` / `verify.sh` / `backup.sh` — provision, smoke-test, backup.

`APP_ENV=development` is intentional: `internal/platform/config/config.go`
rejects `production` without `https://` + `COOKIE_SECURE=true`, which needs a
domain. This bundle is HTTP-only and temporary.

## Deploy (on the VPS, from repo root)

```bash
git clone <repo> /opt/pmt-web && cd /opt/pmt-web
cp deploy/vps-temp/.env.vps.example deploy/vps-temp/.env.vps
nano deploy/vps-temp/.env.vps   # VPS_IP, POSTGRES_PASSWORD, JWT_SECRET, AUTH_BASIC_PASS
# generate: openssl rand -base64 48  (JWT), openssl rand -base64 24 (passwords)
# API_PORT defaults to 8090; set only if occupied on the box.

bash deploy/vps-temp/deploy.sh
bash deploy/vps-temp/verify.sh
```

Shared box (`95.211.43.93` also runs trading): the bundle only adds its own
containers, volumes (`<project>_postgres_data`, `<project>_files_data`),
ufw allow rules (additive), and one cron line. It never restarts, edits, or
prunes anything else. API host port 8090 avoids caddy (:8080) and hermes
(:8000/:8501).

Access links (after deploy prints your IP):

- Frontend: `http://<VPS_IP>/`
- API ready: `http://<VPS_IP>:8090/health/ready` (Basic Auth from `.env.vps`)
- Mailpit: `http://<VPS_IP>:8025/` — verification emails live here

First user flow: register in frontend → open Mailpit → copy token →
`POST /v1/auth/verify-email` (or use the app UI) → login.

## Useful commands

```bash
# logs
docker compose --env-file deploy/vps-temp/.env.vps \
  -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml \
  --profile app logs -f api migrate frontend

# restart after .env.vps change (frontend needs rebuild if VPS_IP changed)
docker compose --env-file deploy/vps-temp/.env.vps \
  -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml \
  --profile app up --build -d

# backup (keep db.sql + files.tar.gz together)
bash deploy/vps-temp/backup.sh

# teardown (keeps volumes; add -v to destroy data)
docker compose --env-file deploy/vps-temp/.env.vps \
  -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml \
  --profile app down
```

## Lock down after testing

- `sudo ufw delete allow 8025/tcp` once email flow is confirmed.
- Do not put real student data here (HTTP-only, temporary secrets).
- Final production needs: domain + TLS, `APP_ENV=production`,
  `COOKIE_SECURE=true`, real SMTP, least-privilege DB user, off-VPS backups.
  Delete or replace this folder at that point.
