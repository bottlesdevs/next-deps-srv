# Dependencies

A dependency is a supporting package that does not occupy a component slot. Examples include redistributable runtimes, fonts, libraries, and data packages needed by a component or application.

Dependencies are published at:

```text
GET /api/v1/catalog/dependencies
```

## When to use a dependency

Publish an entry as a dependency when all of these statements are true:

- it does not replace a bottle component role;
- more than one package of the same general type may be installed;
- consumers identify it by name or UUID rather than a slot.

Use a [component](components.md) when the entry must occupy one of the supported bottle slots.

## Complete dependency submission

This example submits a Windows redistributable package. Replace the URL and digest with values calculated from the real file.

```bash
curl -fsS \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "kind": "dependency",
    "name": "vcredist-2022",
    "version": "14.40.33810",
    "category": "runtime",
    "description": "Microsoft Visual C++ runtime package.",
    "license": "Proprietary",
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
        },
        "steps": []
      }
    ]
  }' \
  https://bottles-next-deps.bromb.in/api/v1/deps
```

Do not send `slot` for a dependency. The server rejects dependency entries that contain one.

## Published dependency entry

After approval and a successful build, the public dependency catalog contains an entry like this:

```json
{
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
      },
      "steps": []
    }
  ]
}
```

Dependency entries never contain `slot`.

## Dependencies with requirements

A dependency can reference entries in either catalog. This example requires a named dependency and one exact entry:

```json
{
  "kind": "dependency",
  "name": "game-runtime-pack",
  "version": "1",
  "artifacts": [
    {
      "url": "https://downloads.example.com/game-runtime-pack.tar.xz",
      "file_name": "game-runtime-pack.tar.xz",
      "checksum": {
        "algorithm": "sha256",
        "value": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
      },
      "steps": []
    }
  ],
  "requirements": [
    { "name": "vcredist-2022" },
    { "id": "7c478786-c11f-4f31-b18c-87063ec72f3d" }
  ]
}
```

A dependency may also require a component slot:

```json
{
  "slot": "runner"
}
```

The registry accepts the selector but does not verify that a matching entry exists at submission time. The public resolver rejects the graph if the requirement is still missing when requested. Local resolvers must perform the same check.

## Client selection

Use the public resolver when the most recently updated built dependency is the desired default:

```json
{
  "dependencies": [
    { "name": "vcredist-2022", "version": "14.40.33810" }
  ],
  "platform": {
    "os": "windows",
    "arch": "x86_64"
  }
}
```

Send this body to `POST /api/v1/resolve` for the ordered entry graph or `POST /api/v1/download-batch` for its flattened artifacts.

For local resolution, a client should:

1. Read the dependency catalog.
2. Find candidates by UUID or name.
3. Choose a version using its own policy.
4. Resolve requirements across both catalogs.
5. Select artifacts matching the target platform.
6. Download each artifact and verify its checksum.

When a root selector has no `version`, the resolver chooses the most recently updated built match. Version strings are opaque. The server does not sort them as SemVer and does not support ranges such as `>=1.2`.

## Platform-independent artifacts

Omit `platform` when an artifact does not declare an OS or architecture restriction:

```json
{
  "url": "https://downloads.example.com/common-data.zip",
  "file_name": "common-data.zip",
  "checksum": {
    "algorithm": "sha256",
    "value": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  },
  "steps": []
}
```

Clients may include an unscoped artifact for any target. A partial platform object is invalid.

## Validation failures

Dependency submissions are rejected when:

- `kind` is not `dependency`;
- a `slot` field is present;
- `name` or `version` is empty;
- no artifacts are present;
- an artifact has an invalid checksum or partial platform;
- a requirement contains zero or multiple selectors.
