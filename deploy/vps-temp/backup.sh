#!/usr/bin/env bash
# Consistent temp backup: pg_dump + files volume. Run from repo root on the VPS.
# Output: deploy/vps-temp/backups/<timestamp>/
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
ENV_FILE="deploy/vps-temp/.env.vps"
OUT="deploy/vps-temp/backups/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUT"

echo "==> pg_dump..."
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml exec -T postgres \
  pg_dump -U lms lms > "$OUT/db.sql"

echo "==> files volume..."
# Scoped to this compose project on purpose: only the PMT files volume may
# ever be named here. Never list or print other volumes on the shared box.
PROJ="$(basename "$ROOT")"
FILES_VOL="$(docker volume ls --format '{{.Name}}' | grep -x "${PROJ}_files_data" | head -1)"
if [[ -z "$FILES_VOL" ]]; then echo "ERROR: no *_files_data volume found"; exit 1; fi
echo "using volume $FILES_VOL"
docker run --rm -v "$FILES_VOL:/data/files:ro" -v "$ROOT/$OUT:/backup" alpine \
  tar -czf /backup/files.tar.gz -C /data/files .

ls -lh "$OUT"
echo "Backup in $OUT. Keep db.sql + files.tar.gz together (metadata references immutable files)."
