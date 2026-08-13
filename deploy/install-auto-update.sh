#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "Run this installer as root" >&2
  exit 1
fi

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
install -m 0755 "$root_dir/deploy/next-deps-srv-update" /usr/local/sbin/next-deps-srv-update
install -m 0644 "$root_dir/deploy/systemd/next-deps-srv-update.service" /etc/systemd/system/next-deps-srv-update.service
install -m 0644 "$root_dir/deploy/systemd/next-deps-srv-update.timer" /etc/systemd/system/next-deps-srv-update.timer
systemctl daemon-reload
systemctl enable --now next-deps-srv-update.timer
systemctl start next-deps-srv-update.service
