# Deployment and automatic updates

Production runs as a native Linux binary managed by systemd. Containers are not required.

## Release artifacts

Pushing a tag that starts with `v` runs `.github/workflows/release.yml`. The workflow runs the Go tests, builds the Vue frontend, builds a static Linux amd64 binary, and publishes these GitHub Release assets:

| Asset | Purpose |
| --- | --- |
| `next-deps-srv-linux-amd64.tar.gz` | Binary, frontend files, and release version. |
| `next-deps-srv-linux-amd64.tar.gz.sha256` | Archive checksum. |
| `next-deps-srv.version` | Small version marker used by the updater. |

Create a release by tagging the commit on `main`:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow creates the GitHub Release after all build steps pass.

## Install the updater

The server layout defaults to:

```text
/home/fab/apps/next-deps-srv/
  next-deps-srv
  VERSION
  data/
  frontend/dist/
```

Install the updater and its timer from a repository checkout:

```bash
sudo ./deploy/install-auto-update.sh
```

The timer checks the latest stable GitHub Release every five minutes. It exits without downloading the archive when `VERSION` already matches the published version.

For a new version, the updater:

1. Downloads the archive and checksum from the public GitHub Release.
2. Verifies the SHA-256 checksum and archive paths.
3. Saves the current binary and frontend under `.rollback/`.
4. Replaces the binary and frontend, then restarts `next-deps-srv.service`.
5. Calls the local catalog endpoint as a health check.
6. Restores the previous release if restart or health verification fails.

The updater never modifies `data/`.

## Operations

Inspect the timer and the most recent update:

```bash
systemctl status next-deps-srv-update.timer
journalctl -u next-deps-srv-update.service -n 100 --no-pager
```

Run a check immediately:

```bash
sudo systemctl start next-deps-srv-update.service
```

Run the updater tests:

```bash
./deploy/next-deps-srv-update_test.sh
```
