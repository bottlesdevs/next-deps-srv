# Components

The component catalog holds every entry that is not a dependency: runners, the runtime tools WineBridge and UMU, and components. A component occupies one slot in a prefix. The slot makes components mutually exclusive: a prefix can select one DXVK implementation for its `dxvk` slot, one VKD3D-Proton implementation for its `vkd3d` slot, and so on. Runners and runtime tools are selected separately and do not occupy a slot.

The component catalog is published at:

```text
GET /api/v1/catalog/components
```

## Kinds

| `kind` | Role | Extra field |
| --- | --- | --- |
| `runner` | Wine or Proton runtime. | `flavor`: `wine` or `proton`. |
| `winebridge` | Bottles WineBridge service executable. | None. |
| `umu` | UMU launcher used with Proton runners. | None. |
| `component` | Replaceable add-on occupying one slot. | `slot`. |

## Supported slots

| Slot | Intended role |
| --- | --- |
| `dxvk` | Direct3D 8, 9, 10, and 11 translation layer. |
| `vkd3d` | Direct3D 12 translation layer. |
| `nvapi` | NVAPI support layer. |
| `latency-flex` | LatencyFleX support component. |

The registry validates kinds, slots, and flavors exactly. A component without a slot and a runner without a flavor are rejected, as is a `slot` or `flavor` on any other kind.

## Runner submission

A runner declares how it is launched:

```json
{
  "kind": "runner",
  "name": "wine",
  "version": "11.0",
  "flavor": "wine",
  "artifacts": [
    {
      "url": "https://downloads.example.com/wine-11.0-x86_64.tar.xz",
      "file_name": "wine-11.0-x86_64.tar.xz",
      "checksum": {
        "algorithm": "sha256",
        "value": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
      },
      "platform": {
        "os": "linux",
        "arch": "x86_64"
      }
    }
  ]
}
```

Runners, WineBridge, and UMU entries do not use `steps`.

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
      { "name": "wine" }
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
  "kind": "component",
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
    { "name": "wine" }
  ]
}
```

Site metadata such as category, description, and license is not included in the catalog document.

## Component requirements

A component may require:

- a component slot, such as `{ "slot": "vkd3d" }`;
- a specific entry UUID, such as `{ "id": "..." }`;
- an entry name, such as `{ "name": "vcredist-2022" }`.

Use a slot when any component in that slot can satisfy the requirement. Runners and runtime tools have no slot, so require them by name or ID. Use an ID when the release must bind to one exact entry. Use a name when the resolver may select the most recently updated built match or when a local client applies its own version policy.

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
2. Find candidates with the requested kind and, for components, slot.
3. Choose one version using its own policy.
4. Resolve the selected entry's requirements.
5. Select artifacts matching the target OS and architecture.
6. Download and verify each selected artifact.

When a root selector has no `version`, the resolver chooses the most recently updated built entry in the slot. Versions are opaque strings and no SemVer ordering is applied. Clients should persist the selected entry UUID and checksums when they need the same result later.

## Validation failures

Component submissions are rejected when:

- `kind` is not `runner`, `winebridge`, `umu`, or `component`;
- a component's `slot` is missing or unknown, or another kind carries a `slot`;
- a runner's `flavor` is missing or unknown, or another kind carries a `flavor`;
- `name` or `version` is empty;
- no artifacts are present;
- an artifact has an invalid checksum or partial platform;
- a requirement contains zero or multiple selectors.
