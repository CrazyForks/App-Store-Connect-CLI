# Asset List Request Timeout Scoping

## Status

Implemented against `ca5f74a8d18261f4d491778eb8f6c5bc0488b205`.

## Problem

`screenshots list` and `video-previews list` are aggregate read commands. Each
command first fetches the sets for one App Store version localization, then
fetches the assets in every returned set.

The current implementations create one `ASC_TIMEOUT` context before the first
request and reuse it for the complete sequence. A healthy set lookup can
therefore consume most of the configured per-request timeout and leave a later
asset lookup with only the remainder. With enough sets, a later request may
start with an already expired context even though it never received its own
configured request budget.

## Command taxonomy and help contract

- `screenshots list` is an existing stable read command. The experimental
  annotation on the top-level screenshots group applies to local capture and
  framing, not this App Store Connect list operation.
- `video-previews list` is an existing stable read command.
- Both commands continue to require `--version-localization` and support the
  shared `--output` and `--pretty` flags.
- Command names, help text, flags, defaults, validation, and exit semantics do
  not change.

## API contract

The offline OpenAPI snapshot defines four independent GET operations:

- `GET /v1/appStoreVersionLocalizations/{id}/appScreenshotSets`
- `GET /v1/appScreenshotSets/{id}/appScreenshots`
- `GET /v1/appStoreVersionLocalizations/{id}/appPreviewSets`
- `GET /v1/appPreviewSets/{id}/appPreviews`

The commands retain their existing response types and aggregation order. No
request path, query parameter, response schema, pagination behavior, or
mutation is added.

## Timeout and output contract

Repository documentation defines `ASC_TIMEOUT` as a per-request timeout. Each
of the GET operations above must therefore receive a fresh timeout derived
from the command's original parent context. The parent context still cancels
the whole command when its caller cancels or supplies an outer deadline.

JSON, table, and Markdown output remain unchanged: the localization ID and
sets are emitted in API order, with each set containing its fetched assets.
Errors keep their existing command and set-specific prefixes.

## Implementation

Create and cancel a fresh `shared.ContextWithTimeout` context immediately
around the set lookup and around each child-asset lookup. Keep the requests
serial and leave the surrounding command structure unchanged.

## Test plan

RED on the exact base:

- Exercise each complete CLI command through a deterministic HTTP transport.
- Give every request a delay that is below `ASC_TIMEOUT`, while the two delays
  together exceed it.
- Assert the command succeeds, makes exactly the set and child-asset GETs, and
  emits the expected aggregate. The current shared context makes the child GET
  expire, so this test fails before the implementation change.

GREEN:

- Re-run the focused command tests for both asset types.
- Build the CLI and repeat both help paths plus required-flag exit behavior
  under isolated home and profile state.
- Run formatting, documentation, lint, and the full test suite.
- If local credentials are available, run read-only list discovery against the
  disposable app. No App Store Connect mutation is needed for this repair.

## Compatibility and failure modes

This only restores the documented timeout budget for independent requests.
Individual requests still fail at `ASC_TIMEOUT`; caller cancellation and outer
deadlines still stop the workflow; authentication, API errors, and partial
fetches still return errors rather than incomplete success output. Total
command duration may continue to exceed `ASC_TIMEOUT`, as expected for a
multi-request command.

## Alternatives considered

- Increase the default timeout: rejected because it only hides the shared
  budget and still fails as the number of sets grows.
- Treat `ASC_TIMEOUT` as one whole-command deadline: rejected because the
  documented contract is per request and other multi-request commands renew
  it at request boundaries.
- Parallelize set lookups: rejected because concurrency changes ordering,
  cancellation, and load behavior beyond the bug being fixed.
