#!/bin/sh
# One-time (idempotent) setup of the schlift.com VM for ppp; `make setup-vm` runs
# it there as root after copying ppp.service and ppp.caddyfile to /tmp. The Maps
# API key goes in /etc/ppp.env (make setup-vm copies it from .env). Afterwards
# `make deploy` installs the build.
set -eu
id ppp >/dev/null 2>&1 || useradd --system --home-dir /opt/ppp --shell /sbin/nologin ppp
install -d -m 755 /opt/ppp
install -d -o ppp -g ppp -m 755 /opt/ppp/db
install -m 644 /tmp/ppp.service /etc/systemd/system/ppp.service
install -d -m 755 /etc/caddy/Caddyfile.d
install -m 644 /tmp/ppp.caddyfile /etc/caddy/Caddyfile.d/ppp.caddyfile
rm -f /tmp/ppp.service /tmp/ppp.caddyfile
out=$(caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile 2>&1) || { echo "$out"; exit 1; }
systemctl reload caddy
systemctl daemon-reload
systemctl enable ppp >/dev/null 2>&1
echo "ppp set up; run make deploy"
