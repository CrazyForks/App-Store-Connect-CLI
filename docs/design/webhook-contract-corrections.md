# Webhook contract corrections

## Placement and command shape

This change stays within the existing public `asc webhooks` and deprecated
`asc marketplace webhooks` command families. Public delivery listing remains:

```text
asc webhooks deliveries --webhook-id "WEBHOOK_ID" [--created-after TIMESTAMP] [--created-before TIMESTAMP]
```

Both date flags are optional and may be combined. Public webhook create and
update continue to accept comma-separated `--events`, but only values from the
documented `WebhookEventType` enum are accepted. The unsupported marketplace
instance `view` command and client method are removed; the supported read
workflow is collection-level `list`.

## API contract

`GET /v1/webhooks/{id}/deliveries` has two independent optional array query
parameters: `filter[createdDateGreaterThanOrEqualTo]` and
`filter[createdDateLessThan]`. Its response is a webhook-delivery collection.
Neither filter is required or mutually exclusive.

Webhook create and update payloads use the exact `WebhookEventType` enum from
the create/update request schema. Invalid examples such as
`SUBSCRIPTION.CREATED` must not be sent to Apple.

`/v1/marketplaceWebhooks/{id}` exposes only `PATCH` and `DELETE` in the current
OpenAPI snapshot. A live read-only `GET` probe also returns `GET_INSTANCE` as
not allowed, so the existing mocked instance GET is not an Apple operation.
Current generated clients in App Store Connect Swift SDK and Bagbutik likewise
offer collection `GET` plus create, update, and delete, but no instance `GET`.
Fastlane and Expo/EAS do not provide an alternate App Store Connect marketplace
webhook detail workflow.

## CLI behavior and compatibility

Successful commands keep their existing TTY-aware output behavior and write
data to stdout. Validation errors use stderr with usage exit code 2. Delivery
date flags become more permissive, which is backward compatible. Event
validation rejects previously accepted invalid values before authentication or
HTTP side effects. Marketplace `view` never mapped to a supported Apple
operation and is removed with its obsolete client method, help, and mock.

## RED-GREEN and verification

- Replace delivery validation tests that encode required/mutually-exclusive
  filters with zero-, one-, and two-filter HTTP query assertions.
- Add create and update CLI tests proving invalid event values fail locally and
  valid enum values are normalized.
- Remove the marketplace instance-GET mock and exercise the supported
  collection list plus instance update/delete neighbors.
- Regenerate command documentation, build `/tmp/asc`, and verify stdout,
  stderr, and exit codes.
- Re-run safe live reads with the explicit file-backed profile. Use a
  deliberately nonexistent marketplace webhook ID and record the rejected
  method/path semantics; do not mutate any real resource.
- Run focused tests, adjacent packages, formatting, documentation checks,
  lint, and the full test suite.

Edge cases include pagination URLs (whose embedded query remains authoritative),
empty comma-separated event input, case normalization, duplicate date values,
API errors, and a marketplace account with no collection entries.

## Alternatives

Retaining marketplace `view` based only on its mock would preserve a command
that Apple explicitly rejects. Direct removal is appropriate because history
contains no evidence it ever worked, the public command was built on an
invented operation, and there is no supported workflow behind the surface.

Client-side filtering marketplace webhook collections by ID could imitate a
view command, but it changes network and pagination semantics and provides no
instance endpoint guarantees. The supported `list` command is the clearer
migration target.
