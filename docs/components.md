# Components

A component is an add-on that occupies one named slot in a bottle. The slot makes component roles mutually exclusive. A bottle can select one runner for its `runner` slot, one DXVK implementation for its `dxvk` slot, and so on.

Components are published at:

```text
GET /api/v1/catalog/components
```

## Supported slots

| Slot | Intended role |
| --- | --- |
| `winebridge` | Wine integration bridge. |
| `runner` | Wine or compatible execution runtime. |
| `umu` | UMU launcher runtime. |
| `dxvk` | Direct3D 8, 9, 10, and 11 translation layer. |
| `vkd3d` | Direct3D 12 translation layer. |
| `nvapi` | NVAPI support layer. |
| `latency-flex` | LatencyFleX support component. |

The registry validates slot names exactly. A component submission without a slot is rejected.

## Complete component submission

This example submits one DXVK release with Linux artifacts for two architectures. Replace the URLs and digests with values calculated from the real files.

```bash
curl -fsS \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "kind": "component",
    "name": "dxvk",
    "version": "2.6.1",
    "slot": "dxvk",
    "category": "graphics",
    "description": "Vulkan-based Direct3D translation layer.",
    "license": "Zlib",
    "artifacts": [
      {
        "url": "https://downloads.example.com/dxvk-2.6.1-x86_64.tar.xz",
        "file_name": "dxvk-2.6.1-x86_64.tar.xz",
        "checksum": {
          "algorithm": "sha256",
          "value": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
        },
        "platform": {
          "os": "linux",
          "arch": "x86_64"
        },
        "steps": []
      },
      {
        "url": "https://downloads.example.com/dxvk-2.6.1-aarch64.tar.xz",
        "file_name": "dxvk-2.6.1-aarch64.tar.xz",
        "checksum": {
          "algorithm": "sha256",
          "value": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
        },
        "platform": {
          "os": "linux",
          "arch": "aarch64"
        },
        "steps": []
      }
    ],
    "requirements": [
      { "slot": "runner" }
    ]
  }' \
  https://bottles-next-deps.bromb.in/api/v1/deps
```

The response contains the server-assigned UUID and status `pending_review`.

## Published component entry

After approval and a successful build, the public component catalog contains an entry like this:

```json
{
  "id": "7c478786-c11f-4f31-b18c-87063ec72f3d",
  "name": "dxvk",
  "version": "2.6.1",
  "slot": "dxvk",
  "artifacts": [
    {
      "url": "https://downloads.example.com/dxvk-2.6.1-x86_64.tar.xz",
      "file_name": "dxvk-2.6.1-x86_64.tar.xz",
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
  "requirements": [
    { "slot": "runner" }
  ]
}
```

Site metadata such as category, description, and license is not included in the catalog document.

## Component requirements

A component may require:

- another component slot, such as `{ "slot": "runner" }`;
- a specific entry UUID, such as `{ "id": "..." }`;
- an entry name, such as `{ "name": "vcredist-2022" }`.

Use a slot when any compatible component in that role can satisfy the requirement. Use an ID when the release must bind to one exact entry. Use a name when the resolver may select the most recently updated built match or when a local client applies its own version policy.

## Client selection

Use the public resolver when the most recently updated built component is the desired default:

```json
{
  "components": [
    { "slot": "dxvk", "version": "2.6.1" }
  ],
  "platform": {
    "os": "windows",
    "arch": "x86_64"
  }
}
```

Send this body to `POST /api/v1/resolve` for the ordered entry graph or `POST /api/v1/download-batch` for its flattened artifacts.

For local resolution, a client should:

1. Read the component catalog.
2. Find candidates with the requested slot.
3. Choose one version using its own policy.
4. Resolve the selected entry's requirements.
5. Select artifacts matching the target OS and architecture.
6. Download and verify each selected artifact.

When a root selector has no `version`, the resolver chooses the most recently updated built entry in the slot. Versions are opaque strings and no SemVer ordering is applied. Clients should persist the selected entry UUID and checksums when they need the same result later.

## Validation failures

Component submissions are rejected when:

- `kind` is not `component`;
- `slot` is missing or unknown;
- `name` or `version` is empty;
- no artifacts are present;
- an artifact has an invalid checksum or partial platform;
- a requirement contains zero or multiple selectors.
