#!/usr/bin/env bash
# Smoke-test the temp VPS deploy. Run from repo root (local or VPS).
# Usage: bash deploy/vps-temp/verify.sh [BASE_IP]
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

IP="${1:-}"
if [[ -z "$IP" && -f deploy/vps-temp/.env.vps ]]; then
  set -a; . deploy/vps-temp/.env.vps; set +a
  IP="${VPS_IP:-}"
fi
if [[ -z "$IP" ]]; then echo "Usage: bash deploy/vps-temp/verify.sh [VPS_IP]"; exit 1; fi
AUTH_ARGS=()
API_PORT=8090
if [[ -f deploy/vps-temp/.env.vps ]]; then
  set -a; . deploy/vps-temp/.env.vps; set +a
  AUTH_ARGS=(-u "${AUTH_BASIC_USER:-admin}:${AUTH_BASIC_PASS:-}")
  API_PORT="${API_PORT:-8090}"
fi

echo "==> containers"
# NOTE: the fallback is scoped to this compose project on purpose — a plain
# `docker ps` would print EVERY container on the shared box (including the
# trading stack) into logs that stream back to GitHub. Never broaden this.
PROJ="$(basename "$ROOT")"
docker compose --env-file deploy/vps-temp/.env.vps -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml ps 2>/dev/null || docker ps --filter "label=com.docker.compose.project=$PROJ" --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'

echo "==> GET http://$IP/ (frontend)"
curl -sf "http://$IP/" -o /dev/null -w "frontend %{http_code}\n"

echo "==> GET http://$IP:$API_PORT/health/live + /ready (basic auth)"
curl -sf "${AUTH_ARGS[@]}" "http://$IP:$API_PORT/health/live" && echo " live OK"
curl -sf "${AUTH_ARGS[@]}" "http://$IP:$API_PORT/health/ready" && echo " ready OK"

echo "==> proxy via :80"
curl -sf "${AUTH_ARGS[@]}" "http://$IP/health/ready" && echo " proxy ready OK"
curl -sf "http://$IP/v1/public/gallery?limit=1" && echo "" && echo " public gallery OK"

echo "==> Mailpit"
curl -sf "http://$IP:8025/" -o /dev/null -w "mailpit %{http_code}\n" || echo "(mailpit not reachable — check firewall/compose)"

echo "ALL CHECKS DONE"
