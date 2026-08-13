# Publishing entries

Publishing has two sides. A contributor describes an entry and its source artifacts. A moderator or administrator reviews the request and starts the build. The entry appears in a public catalog only after the build succeeds.

## Roles

| Action | Required role |
| --- | --- |
| Read public catalogs | None |
| Submit an entry | `contributor`, `mod`, or `admin` |
| Review an entry | `mod` or `admin` |
| Inspect and trigger jobs | `admin` |
| Delete an entry reference | `admin` |

New accounts created through the registration endpoint receive the `contributor` role.

## Authentication

Register an account:

```bash
curl -fsS \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "replace-with-a-private-password"
  }' \
  https://bottles-next-deps.bromb.in/api/v1/auth/register
```

Log in:

```bash
curl -fsS \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "alice",
    "password": "replace-with-a-private-password"
  }' \
  https://bottles-next-deps.bromb.in/api/v1/auth/login
```

Both endpoints return a token:

```json
{
  "token": "..."
}
```

Send it in the `Authorization` header for protected endpoints:

```text
Authorization: Bearer <token>
```

Access tokens expire after 24 hours.

## Submission endpoint

Submit either kind with:

```text
POST /api/v1/deps
```

The server assigns the entry UUID. Any client-supplied `id` is ignored.

Common top-level fields:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `kind` | string | Yes | `component` or `dependency`. |
| `name` | string | Yes | Human-readable package name. |
| `version` | string | Yes | Opaque version string. The server does not enforce SemVer. |
| `slot` | string | Components only | Required for components and forbidden for dependencies. |
| `artifacts` | array | Yes | Must contain at least one artifact. |
| `requirements` | array | No | References other entries by name, slot, or UUID. |
| `license` | string | No | Site metadata, not part of the public catalog entry. |
| `category` | string | No | Site metadata, not part of the public catalog entry. |
| `description` | string | No | Site metadata, not part of the public catalog entry. |

Each artifact has this shape:

```json
{
  "url": "https://downloads.example.com/package.tar.xz",
  "file_name": "package.tar.xz",
  "checksum": {
    "algorithm": "sha256",
    "value": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  },
  "platform": {
    "os": "linux",
    "arch": "x86_64"
  },
  "steps": []
}
```

Artifact rules:

- `url` and `file_name` must not be empty.
- `checksum.algorithm` must be `sha256` or `sha512`.
- The digest must be lowercase hexadecimal with the exact algorithm length.
- `platform` is optional, but `os` and `arch` must both be present when it is used.
- `steps` is an optional JSON array. The registry stores it without interpreting its contents.

Supported platform values:

| Field | Values |
| --- | --- |
| `os` | `linux`, `mac-os`, `windows` |
| `arch` | `x86`, `x86_64`, `aarch64` |

## Requirements

Each requirement must contain exactly one selector:

```json
{ "name": "vcredist-2022" }
```

```json
{ "slot": "runner" }
```

```json
{ "id": "7c478786-c11f-4f31-b18c-87063ec72f3d" }
```

Name and ID selectors may resolve to either catalog. Slot selectors resolve only to components. The server validates the selector shape but does not check that the referenced entry already exists.

## Review and build lifecycle

The initial response has status `pending_review`. A moderator can approve it:

```text
POST /api/v1/deps/{id}/approve
```

Approval creates a build job. The worker processes each artifact as follows:

1. Download the source URL.
2. Compute and compare the declared checksum.
3. Attempt to store and index the source archive. A source-index error is logged without stopping extraction.
4. Extract nested archives, up to the worker extraction limit.
5. Store and index extracted files as revisions.
6. Mark the entry `built` after every artifact succeeds.

The queue runs up to three builds at the same time. One failed artifact fails the whole job. A failed entry returns to `approved`, so an administrator can inspect the job and trigger it again.

The published document keeps the submitted artifact URL. It does not replace that URL with an internal mirror URL.

## Deleting a catalog reference

An administrator can remove a component or dependency record:

```text
DELETE /api/v1/admin/deps/{id}
```

The endpoint deletes only the catalog reference. Indexed bucket files, file revisions, and stored object bytes are retained. This is useful when an entry should no longer be published but its indexed history must remain available.

## Status reference

| Status | Meaning | Public catalog |
| --- | --- | --- |
| `pending_review` | Waiting for moderator review. | No |
| `approved` | Accepted, or ready for another build attempt. | No |
| `building` | A worker is processing the artifacts. | No |
| `built` | Every artifact passed and indexing completed. | Yes |
| `rejected` | A moderator rejected the submission. | No |

## Publishing a component or dependency

Use the kind-specific guides for complete requests:

- [Publishing components](components.md)
- [Publishing dependencies](dependencies.md)
