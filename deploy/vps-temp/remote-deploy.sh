#!/usr/bin/env bash
# Forced-command entrypoint for the PMT deploy key (pmtdeploy user).
#
# Install on the VPS once (as root):
#   mkdir -p /home/pmtdeploy/.ssh && chmod 700 /home/pmtdeploy/.ssh
#   echo 'command="/opt/pmt-web/deploy/vps-temp/remote-deploy.sh",no-agent-forwarding,no-X11-forwarding,no-pty <PASTE pmt_vps_deploy.pub HERE>' \
#     > /home/pmtdeploy/.ssh/authorized_keys
#   chmod 600 /home/pmtdeploy/.ssh/authorized_keys
#   chown -R pmtdeploy:pmtdeploy /home/pmtdeploy/.ssh
#
# Security properties (this is what isolates the trading agent):
# - ANY ssh session with that key runs ONLY this script — no shell, no scp,
#   no port forwarding, no pty. SSH_ORIGINAL_COMMAND is ignored on purpose.
# - pmtdeploy is NOT root, has NO sudo, and owns ONLY /opt/pmt-web, so the key
#   can pull PMT code and restart PMT containers — and nothing else.
# - VPS secrets (.env.vps) are never printed and never leave the VPS.
# - hermes-* containers / /opt/hermes / the 95.211.126.202 box are untouched.
set -euo pipefail

# Deliberately ignore any client-supplied command (forced command always wins).
unset SSH_ORIGINAL_COMMAND 2>/dev/null || true

echo "==> $(date -u +%FT%TZ) PMT auto-deploy triggered as user $(whoami)"
cd /opt/pmt-web

echo "==> Pulling latest main (fast-forward only)..."
git pull --ff-only

echo "==> Deploying..."
bash deploy/vps-temp/deploy.sh

echo "==> Smoke-testing..."
bash deploy/vps-temp/verify.sh

echo "AUTO-DEPLOY DONE"
