#!/usr/bin/env bash
# Temp VPS deploy — Ubuntu + Docker, IP-only HTTP. Run from repo root on the VPS.
#   bash deploy/vps-temp/deploy.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
ENV_FILE="deploy/vps-temp/.env.vps"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE. Copy the template first:"
  echo "  cp deploy/vps-temp/.env.vps.example $ENV_FILE"
  echo "  nano $ENV_FILE   # set VPS_IP, POSTGRES_PASSWORD, JWT_SECRET, AUTH_BASIC_PASS"
  exit 1
fi
set -a; . "./$ENV_FILE"; set +a
API_PORT="${API_PORT:-8090}"

for v in VPS_IP POSTGRES_PASSWORD JWT_SECRET AUTH_BASIC_PASS; do
  if [[ -z "${!v:-}" || "${!v}" == REPLACE_* ]]; then
    echo "ERROR: $v is not set in $ENV_FILE"
    exit 1
  fi
done
if ((${#JWT_SECRET} < 32)); then echo "ERROR: JWT_SECRET must be >= 32 chars"; exit 1; fi

echo "==> VPS_IP=$VPS_IP"

# Docker (Ubuntu) if missing
if ! command -v docker >/dev/null 2>&1; then
  echo "==> Installing Docker..."
  sudo apt-get update -y
  sudo apt-get install -y ca-certificates curl gnupg
  sudo install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  sudo chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
  sudo apt-get update -y
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  sudo usermod -aG docker "$USER" || true
fi
docker compose version

# Firewall (additive only — never touch existing rules): open the PMT ports.
# DB stays localhost-only. 22/443 already allowed on the shared box.
if command -v ufw >/dev/null 2>&1; then
  sudo ufw allow 80/tcp || true
  sudo ufw allow "$API_PORT/tcp" || true
  sudo ufw allow 8025/tcp || true
  sudo ufw status || true
fi

echo "==> Validating compose config..."
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml --profile app config >/dev/null

echo "==> Building + starting (postgres, mailpit, migrate, api, frontend)..."
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml --profile app up --build -d

echo "==> Waiting for migration + API..."
for i in $(seq 1 30); do
  if docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml ps --status running | grep -q api; then break; fi
  sleep 3
done
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml ps
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml logs migrate --tail 30 || true

echo "==> Health check (Basic Auth from .env.vps)..."
sleep 5
curl -sf -u "$AUTH_BASIC_USER:$AUTH_BASIC_PASS" "http://localhost:$API_PORT/health/live" && echo " live OK"
curl -sf -u "$AUTH_BASIC_USER:$AUTH_BASIC_PASS" "http://localhost:$API_PORT/health/ready" && echo " ready OK"
curl -sf "http://localhost/" -o /dev/null -w "frontend %{http_code}\n"

# Cron for background jobs (notifications, expired uploads, session cleanup).
CRON="*/5 * * * * cd $ROOT && docker compose --env-file $ENV_FILE -f docker-compose.yml -f deploy/vps-temp/docker-compose.vps.yml run --rm jobs >> /var/log/pmt-jobs.log 2>&1"
if ! crontab -l 2>/dev/null | grep -q "deploy/vps-temp.* run --rm jobs"; then
  (crontab -l 2>/dev/null; echo "$CRON") | crontab -
  echo "==> Installed jobs cron (every 5 min)"
fi

echo ""
echo "DONE. Public links (replace if firewall differs):"
echo "  Frontend: http://$VPS_IP/"
echo "  API:      http://$VPS_IP:$API_PORT/health/ready  (user: $AUTH_BASIC_USER)"
echo "  Mailpit:  http://$VPS_IP:8025/  (lock down after testing)"
echo "Run: bash deploy/vps-temp/verify.sh"
