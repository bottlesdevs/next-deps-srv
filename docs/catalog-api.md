# Catalog API

The catalog API is the public interface used by Bottles clients and other consumers. It publishes built entries as JSON. Reads do not require authentication.

## Base URL

```text
https://bottles-next-deps.bromb.in/api/v1
```

## Endpoints

| Method | Path | Response |
| --- | --- | --- |
| `GET` | `/catalog/components` | Component entries with required slots. |
| `GET` | `/catalog/dependencies` | Dependency entries without slots. |
| `POST` | `/resolve` | Ordered component and dependency graph. |
| `POST` | `/download-batch` | Flattened, deduplicated artifacts for a resolved graph. |
| `GET` | `/deps/{id}/files` | Indexed files produced by one entry. |
| `GET` | `/files/{name}` | Indexed revisions for an exact file name. |
| `GET` | `/files/download/{revision_id}` | Bytes for one indexed revision. |

Catalog and file endpoints may return `429 Too Many Requests` when rate limiting is enabled.

## Catalog document

Both catalogs use this envelope:

```json
{
  "schema_version": 1,
  "entries": [
    {
      "id": "7c478786-c11f-4f31-b18c-87063ec72f3d",
      "name": "example-package",
      "version": "1.0.0",
      "artifacts": [
        {
          "url": "https://downloads.example.com/example-package.tar.xz",
          "file_name": "example-package.tar.xz",
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
      ],
      "requirements": []
    }
  ]
}
```

Clients must check `schema_version` before parsing entries. Version `1` is the only published schema at this time.

## Entry fields

| Field | Type | Meaning |
| --- | --- | --- |
| `id` | UUID string | Server-assigned identity for one published release. |
| `name` | string | Package name. |
| `version` | string | Opaque version value. |
| `slot` | string | Component role. Present only in the component catalog. |
| `artifacts` | array | Download candidates and their verification data. |
| `requirements` | array | Optional references to other catalog entries. |
| `artifacts[].component_root` | string, optional | Subdirectory of the archive's own top-level directory that is the component itself, for archives that wrap it in unrelated packaging (e.g. a macOS runner's Wine layout nested inside an app bundle). Relative, and never containing `..`. |

The server publishes at most one entry for each `name` and `version` pair in each catalog. If multiple built records share the same pair, the most recently updated record wins. Clients must not use array order as a version preference.

## Artifact selection

Each artifact contains a source URL, file name, checksum, optional platform, and optional steps.

Platform matching is exact:

```text
linux/x86_64 != linux/aarch64
windows/x86 != windows/x86_64
```

An artifact without `platform` declares no target restriction. The server does not infer architecture compatibility.

A client should reject a download when:

- the HTTP request fails;
- the calculated digest differs from `checksum.value`;
- the algorithm is unsupported by the client;
- required installation steps are unknown to the client.

Checksums are lowercase and case-sensitive. Verify the downloaded bytes before extraction or installation.

## Requirement resolution

A requirement contains exactly one of these keys:

| Selector | Search scope | Expected behavior |
| --- | --- | --- |
| `id` | Both catalogs | Match the exact entry UUID. |
| `name` | Both catalogs | Select the most recently updated built match. |
| `slot` | Component catalog | Select the most recently updated built component in that slot. |

The public resolver accepts root selectors and expands their requirements recursively:

```bash
curl -fsS \
  -H 'Content-Type: application/json' \
  -d '{
    "components": [
      { "slot": "dxvk", "version": "2.6.1" }
    ],
    "dependencies": [
      { "name": "vcredist-2022" }
    ],
    "platform": {
      "os": "windows",
      "arch": "x86_64"
    }
  }' \
  https://bottles-next-deps.bromb.in/api/v1/resolve
```

Each root selector must contain exactly one of `id`, `name`, or `slot`. A root component may use any of the three. A root dependency cannot use `slot`. The optional `version` field requires an exact opaque version match. When it is omitted, the most recently updated built match is selected.

The optional `platform` uses the same values as artifact platforms. When supplied, each resolved entry contains only artifacts that either match the target exactly or have no platform restriction. Resolution fails when a selected entry has no compatible artifact.

Example response:

```json
{
  "schema_version": 1,
  "platform": {
    "os": "windows",
    "arch": "x86_64"
  },
  "entries": [
    {
      "kind": "dependency",
      "id": "07f294e7-bbeb-47af-b24d-860f09cff325",
      "name": "vcredist-2022",
      "version": "14.40.33810",
      "artifacts": [
        {
          "url": "https://downloads.example.com/vc_redist.x64.exe",
          "file_name": "vc_redist.x64.exe",
          "checksum": {
            "algorithm": "sha256",
            "value": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
          },
          "platform": {
            "os": "windows",
            "arch": "x86_64"
          }
        }
      ]
    },
    {
      "kind": "component",
      "id": "7c478786-c11f-4f31-b18c-87063ec72f3d",
      "name": "dxvk",
      "version": "2.6.1",
      "slot": "dxvk",
      "artifacts": [
        {
          "url": "https://downloads.example.com/dxvk-2.6.1.tar.xz",
          "file_name": "dxvk-2.6.1.tar.xz",
          "checksum": {
            "algorithm": "sha256",
            "value": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
          },
          "platform": {
            "os": "windows",
            "arch": "x86_64"
          }
        }
      ],
      "requirements": [
        { "name": "vcredist-2022" }
      ]
    }
  ]
}
```

Entries are ordered requirements first and requested roots last. A shared requirement appears once. The server returns `400 Bad Request` for invalid selectors, unknown roots, missing requirements, cycles, or a target with no compatible artifact.

Requirement objects do not carry a version. A name or slot requirement therefore selects the most recently updated built match. Use an ID requirement when an entry must bind to one exact release.

## Download batches

`POST /download-batch` accepts the same body as `/resolve`. It returns the resolved artifacts as one download list:

```json
{
  "schema_version": 1,
  "platform": {
    "os": "windows",
    "arch": "x86_64"
  },
  "downloads": [
    {
      "sources": [
        {
          "id": "07f294e7-bbeb-47af-b24d-860f09cff325",
          "name": "vcredist-2022",
          "version": "14.40.33810",
          "kind": "dependency"
        }
      ],
      "url": "https://downloads.example.com/vc_redist.x64.exe",
      "file_name": "vc_redist.x64.exe",
      "checksum": {
        "algorithm": "sha256",
        "value": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
      },
      "platform": {
        "os": "windows",
        "arch": "x86_64"
      },
      "steps": []
    }
  ]
}
```

Identical artifact definitions are emitted once. `sources` lists every resolved entry that declared the artifact. URLs remain the submitted source URLs. The server does not download the batch on behalf of the client, so the client must fetch each URL and verify the declared checksum.

A client that needs a different selection policy can resolve locally:

1. Fetch both documents and verify their schema versions.
2. Build indexes by ID, name, and component slot.
3. Select the requested root entries.
4. Resolve requirements recursively.
5. Reject missing references and dependency cycles.
6. Apply the client's version policy when a name or slot has multiple candidates.
7. Select artifacts for the requested platform.
8. Deduplicate artifacts before scheduling downloads.

## Minimal JavaScript client

```js
const baseURL = 'https://bottles-next-deps.bromb.in/api/v1'

async function getCatalog(kind) {
  const response = await fetch(`${baseURL}/catalog/${kind}`)
  if (!response.ok) {
    throw new Error(`catalog request failed: ${response.status}`)
  }

  const catalog = await response.json()
  if (catalog.schema_version !== 1) {
    throw new Error(`unsupported schema: ${catalog.schema_version}`)
  }
  return catalog.entries
}

const [components, dependencies] = await Promise.all([
  getCatalog('components'),
  getCatalog('dependencies'),
])

const target = { os: 'linux', arch: 'x86_64' }

function supportsTarget(artifact) {
  if (!artifact.platform) return true
  return artifact.platform.os === target.os && artifact.platform.arch === target.arch
}

const downloads = [...components, ...dependencies]
  .flatMap(entry => entry.artifacts)
  .filter(supportsTarget)
```

This example only fetches and filters artifacts. A production client must resolve requirements, choose versions, verify checksums, and handle unsupported steps.

## Downloading indexed files

The build worker stores the source archive and each extracted file as indexed revisions. Look up a file by its exact name:

```bash
curl -fsS \
  https://bottles-next-deps.bromb.in/api/v1/deps/ENTRY_ID/files
```

This returns the files attributed to that entry, their revision counts, and a download URL for the latest revision produced by that entry. It does not require the entry's revision to be the newest global revision for that file name.

Look up all revisions of a file by its exact name:

```bash
curl -fsS \
  https://bottles-next-deps.bromb.in/api/v1/files/example.dll
```

Example response:

```json
{
  "file": {
    "id": "...",
    "name": "example.dll",
    "latest_rev_id": "..."
  },
  "revisions": [
    {
      "id": "...",
      "revision_num": 1,
      "hash": "...",
      "size_bytes": 123456,
      "source_dep": "...",
      "archive_url": "https://downloads.example.com/example-package.tar.xz",
      "download_url": "/api/v1/files/download/..."
    }
  ]
}
```

Resolve the relative `download_url` against the API origin, then request it:

```bash
curl -fL -o example.dll \
  https://bottles-next-deps.bromb.in/api/v1/files/download/REVISION_ID
```

File lookup is global and keyed by file name. It is separate from catalog artifact selection.

## Current limits

The current API does not provide:

- version constraints or version ordering;
- a generated lockfile;
- a ZIP containing a resolved batch;
- catalog URLs rewritten to the indexed server copies;
- cache validators such as `ETag` or `Last-Modified`.

Consumers should cache successful catalog responses locally, respect rate limits, and retain the selected entry IDs and checksums when they need reproducible installs.
