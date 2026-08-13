# Bottles Next dependency catalogs

This directory documents the public catalogs exposed by `next-deps-srv`, the publication workflow, and the rules a client must follow when it consumes components or dependencies.

The production API is available at:

```text
https://bottles-next-deps.bromb.in/api/v1
```

## Start here

- [Publishing entries](publishing.md): accounts, submission, review, build, and publication.
- [Catalog API](catalog-api.md): public endpoints, shared schema, client resolution, and downloads.
- [Components](components.md): bottle slots, component requirements, and complete payloads.
- [Dependencies](dependencies.md): non-slot packages, dependency requirements, and complete payloads.

## Core model

The registry publishes two catalog documents:

| Catalog | Endpoint | Purpose |
| --- | --- | --- |
| Components | `GET /catalog/components` | Add-ons that occupy a named slot in a bottle. |
| Dependencies | `GET /catalog/dependencies` | Supporting packages that do not occupy a slot. |

Both endpoints are public and return the same document envelope:

```json
{
  "schema_version": 1,
  "entries": []
}
```

An entry becomes visible only after approval and a successful build. The build service downloads every declared artifact, checks its digest, extracts it, and indexes the resulting files.

```text
submission -> pending_review -> approved -> building -> built -> catalog
```

A rejected entry is not published. A failed build returns the entry to `approved` and keeps it out of the public catalog.

## Component and dependency differences

| Rule | Component | Dependency |
| --- | --- | --- |
| `kind` on submission | `component` | `dependency` |
| `slot` | Required | Forbidden |
| Catalog | `/catalog/components` | `/catalog/dependencies` |
| May satisfy a slot requirement | Yes | No |
| May satisfy a name or ID requirement | Yes | Yes |

Artifacts, checksums, platforms, versions, and requirements use the same schema in both catalogs.

## Quick check

```bash
curl -fsS https://bottles-next-deps.bromb.in/api/v1/catalog/components
curl -fsS https://bottles-next-deps.bromb.in/api/v1/catalog/dependencies
```

No bearer token is required for catalog reads. Public catalog and file endpoints may be rate-limited by server configuration.

## Current client contract

The server exposes a resolver and a flattened download batch. A consuming application can:

1. Submit component and dependency selectors to `POST /resolve`.
2. Use the dependency-first entry graph returned by the server.
3. Submit the same request to `POST /download-batch` when it needs a deduplicated artifact list.
4. Download artifact URLs and verify their checksums.
5. Interpret only the `steps` values that it supports.

Clients that need a different version policy can still read both catalog documents and resolve entries locally. See [Catalog API](catalog-api.md) for the complete contract and current limits.
