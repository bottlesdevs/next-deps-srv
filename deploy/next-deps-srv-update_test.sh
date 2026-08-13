#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
test_dir=$(mktemp -d -t next-deps-srv-update-test.XXXXXX)
trap 'rm -rf -- "$test_dir"' EXIT

app_dir="$test_dir/app"
release_dir="$test_dir/releases/latest/download"
mkdir -p "$app_dir/data" "$app_dir/frontend/dist" "$release_dir" "$test_dir/bin"
printf '#!/usr/bin/env bash\nprintf old\n' > "$app_dir/next-deps-srv"
chmod 0755 "$app_dir/next-deps-srv"
printf 'old-ui\n' > "$app_dir/frontend/dist/index.html"
printf 'v0.0.0\n' > "$app_dir/VERSION"
printf 'keep-me\n' > "$app_dir/data/sentinel"
printf '#!/usr/bin/env bash\nexit 0\n' > "$test_dir/bin/systemctl"
chmod 0755 "$test_dir/bin/systemctl"
printf 'healthy\n' > "$test_dir/health"

publish_fixture() {
  version=$1
  binary_text=$2
  fixture="$test_dir/fixture"
  rm -rf -- "$fixture"
  mkdir -p "$fixture/frontend/dist"
  printf '#!/usr/bin/env bash\nprintf %s\n' "$binary_text" > "$fixture/next-deps-srv"
  chmod 0755 "$fixture/next-deps-srv"
  printf '%s-ui\n' "$binary_text" > "$fixture/frontend/dist/index.html"
  printf '%s\n' "$version" > "$fixture/VERSION"
  tar -C "$fixture" -czf "$release_dir/next-deps-srv-linux-amd64.tar.gz" .
  sha256sum "$release_dir/next-deps-srv-linux-amd64.tar.gz" > "$release_dir/next-deps-srv-linux-amd64.tar.gz.sha256"
  printf '%s\n' "$version" > "$release_dir/next-deps-srv.version"
}

run_update() {
  NEXT_DEPS_APP_DIR="$app_dir" \
  NEXT_DEPS_RELEASE_BASE_URL="file://$test_dir/releases" \
  NEXT_DEPS_HEALTH_URL="${NEXT_DEPS_HEALTH_URL:-file://$test_dir/health}" \
  NEXT_DEPS_SERVICE_OWNER=$(id -un) \
  NEXT_DEPS_SERVICE_GROUP=$(id -gn) \
  NEXT_DEPS_LOCK_FILE="$test_dir/update.lock" \
  NEXT_DEPS_SYSTEMCTL_BIN="$test_dir/bin/systemctl" \
  NEXT_DEPS_HEALTH_ATTEMPTS=1 \
  NEXT_DEPS_HEALTH_DELAY=0 \
  "$root_dir/deploy/next-deps-srv-update"
}

publish_fixture v0.1.0 new
run_update
test "$("$app_dir/next-deps-srv")" = new
test "$(tr -d '\r\n' < "$app_dir/VERSION")" = v0.1.0
test "$(tr -d '\r\n' < "$app_dir/frontend/dist/index.html")" = new-ui
test "$(tr -d '\r\n' < "$app_dir/data/sentinel")" = keep-me

run_update

publish_fixture v0.2.0 broken
printf '0%.0s' {1..64} > "$release_dir/next-deps-srv-linux-amd64.tar.gz.sha256"
printf '  next-deps-srv-linux-amd64.tar.gz\n' >> "$release_dir/next-deps-srv-linux-amd64.tar.gz.sha256"
if run_update; then
  echo "Expected a checksum mismatch to reject the release" >&2
  exit 1
fi
test "$("$app_dir/next-deps-srv")" = new

publish_fixture v0.1.1 linked
ln -s /etc/passwd "$test_dir/fixture/frontend/dist/passwd"
tar -C "$test_dir/fixture" -czf "$release_dir/next-deps-srv-linux-amd64.tar.gz" .
sha256sum "$release_dir/next-deps-srv-linux-amd64.tar.gz" > "$release_dir/next-deps-srv-linux-amd64.tar.gz.sha256"
if run_update; then
  echo "Expected an archive link to be rejected" >&2
  exit 1
fi
test "$("$app_dir/next-deps-srv")" = new

publish_fixture v0.2.0 broken
if NEXT_DEPS_HEALTH_URL="file://$test_dir/missing-health" run_update; then
  echo "Expected the failed health check to reject the release" >&2
  exit 1
fi
test "$("$app_dir/next-deps-srv")" = new
test "$(tr -d '\r\n' < "$app_dir/VERSION")" = v0.1.0
test "$(tr -d '\r\n' < "$app_dir/frontend/dist/index.html")" = new-ui
test "$(tr -d '\r\n' < "$app_dir/data/sentinel")" = keep-me

printf 'invalid\n' > "$release_dir/next-deps-srv.version"
if run_update; then
  echo "Expected an invalid version to be rejected" >&2
  exit 1
fi

echo "next-deps-srv updater tests passed"
