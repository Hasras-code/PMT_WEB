#!/usr/bin/env bash
# Generate deploy/vps-temp/.env.vps with random secrets. Run on the VPS from repo root:
#   bash deploy/vps-temp/make-env.sh [VPS_IP] [API_PORT]
# Never overwrites an existing file. Never commit the output.
set -euo pipefail
ENV_FILE="deploy/vps-temp/.env.vps"
if [[ -f "$ENV_FILE" ]]; then
  echo "exists: $ENV_FILE (not overwriting)"
  exit 0
fi
VPS_IP="${1:-95.211.43.93}"
API_PORT="${2:-8090}"
JWT="$(openssl rand -base64 48)"
DBPW="$(openssl rand -base64 32 | tr -dc 'A-Za-z0-9' | head -c 24)"
HEALTH="$(openssl rand -base64 32 | tr -dc 'A-Za-z0-9' | head -c 20)"
cat > "$ENV_FILE" <<EOF
VPS_IP=$VPS_IP
API_PORT=$API_PORT
POSTGRES_PASSWORD=$DBPW
JWT_SECRET=$JWT
AUTH_BASIC_USER=admin
AUTH_BASIC_PASS=$HEALTH
TUNNEL_URL=
VITE_API_URL=http://$VPS_IP:$API_PORT
EOF
chmod 600 "$ENV_FILE"
echo "wrote $ENV_FILE"
grep -E '^(VPS_IP|API_PORT|AUTH_BASIC_USER|VITE_API_URL)' "$ENV_FILE"
