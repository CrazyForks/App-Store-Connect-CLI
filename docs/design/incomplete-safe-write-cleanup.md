# Incomplete safe-write cleanup

## Decision and placement

`internal/cli/shared.SafeWriteFileNoSymlink` remains the shared artifact-write
boundary used by asset downloads and certificate CSR generation. When
`overwrite` is false, the helper will continue to create the destination
exclusively, but any callback, sync, or close failure will make a best effort to
close and remove the file created by that invocation. A destination that existed
before the call is never opened, truncated, or removed.

This changes no command registration or public command taxonomy. Current help
and invocation shapes remain unchanged, including `asc screenshots download`
and its default `--overwrite=false` behavior. There is no App Store Connect API
endpoint, request schema, query parameter, or response shape involved.

## Observable behavior and compatibility

Successful writes retain their existing byte count, permissions, stdout, and
stderr behavior. Existing-destination failures retain the current
`output file already exists` error. Callback and sync errors remain discoverable
with `errors.Is`; when closing or removing the incomplete file also fails, the
returned error additionally exposes that cleanup error rather than hiding it.
The overwrite temp-and-backup path is unchanged.

The change is backward compatible for flags, output formats, exit codes, and
scripts. Its only artifact-level difference is that a failed non-overwrite write
does not leave a newly created partial file behind when cleanup succeeds. That
allows a download or certificate-generation retry to use the same destination.
No lifecycle, migration, deprecation, or command-documentation update is
required.

## RED-GREEN and verification

Shared-helper regression tests will first establish RED for:

- a callback that writes partial bytes and then returns a sentinel error;
- deterministic injected sync and close failures on every supported platform;
- preservation of the primary error plus any close or remove cleanup error;
- a successful new-file write; and
- an existing destination whose contents must remain unchanged.

GREEN verification will cover the shared package, asset-download and
certificate consumers, then the repository format, documentation, lint, and
full test gates. A built `/tmp/asc` characterization will exercise a realistic
local write failure if the host offers a deterministic, non-destructive path;
otherwise the injected helper tests are the portable failure proof. No live App
Store Connect mutation is needed.

## Failure modes and alternatives

Cleanup itself can fail, for example when the filesystem refuses a close or
remove. The helper cannot claim the artifact is absent in that case, so it will
return both the original operation error and the cleanup error. The destination
is removed only after this invocation created it exclusively; the
existing-destination branch returns before cleanup is armed.

Writing every non-overwrite artifact to a sibling temporary file and publishing
it with no-replace atomic semantics was considered. Go does not expose one
portable no-replace rename operation, and platform-specific publication would
substantially widen this repair. Reusing the overwrite backup flow was also
rejected because it permits replacement semantics that `overwrite=false` must
not acquire. Narrow failure cleanup preserves the current exclusivity contract
and fixes the retry-blocking partial artifact without changing overwrite logic.
