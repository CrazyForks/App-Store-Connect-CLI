# Scope-Aware CI Selection

## Current behavior

Except for Wall-only changes, every pull request runs the complete formatting,
documentation, lint, test-shard, Windows telemetry, and five-binary build suite.
Website content and Studio changes also pay for those general CLI checks despite
having narrower validation needs.

The protected branch requires three PR check names: `format-and-lint`,
`unit-tests`, and `build`. Any selection scheme must keep those checks present
and must fail closed when change detection is unavailable or ambiguous.

## Proposed behavior

Classify the complete base-to-head changed-file list into one conservative
scope:

| Scope | Selected validation |
| --- | --- |
| Wall source only | Wall source validation |
| Repository documentation only | Formatting and documentation validation |
| Mintlify website content only | Dedicated website validator workflow |
| `apps/studio` only | Dedicated Studio frontend, Go test, and build workflow |
| Telemetry packages only | Formatting, targeted Linux and Windows tests, and Linux/macOS `go build .` |
| Any general, mixed, or unknown change | Full required suite and native platform builds |

The three required PR jobs always resolve. The required `format-and-lint`
aggregate waits for and verifies every affected Wall, website, Studio, or
general quality job. The test and build aggregates report when no general Go
work is required. A failure in the change detector fails all required
aggregates instead of silently selecting a smaller suite.

Website validation is an Ubuntu reusable workflow. Studio remains a macOS
reusable workflow because its Wails build requires native GUI development
libraries on Linux. General cross-platform compilation continues to use Linux,
macOS, and Windows runners through the runner split from the preceding CI
optimization change, with Darwin-only screenshot tests on the macOS leg.

## Safety boundaries

- Only exact, allowlisted paths receive a reduced scope.
- Mixed specialized areas fall back to the full suite.
- Specialized code plus documentation falls back to the full suite so the
  documentation is not skipped.
- Workflow, build-system, dependency, shared package, and classifier changes
  always receive the full suite.
- Go source is always treated as code, even when it lives under a documentation
  directory.
- The OpenAPI snapshot receives the full suite because Go tests consume it as
  schema-drift input.
- A manual PR workflow dispatch has no changed-file list and therefore receives
  the full suite.

## Verification

- Unit-test path classification, including empty, mixed, and unknown changes.
- Assert workflow ownership, required aggregate jobs, native build runners, and
  release-only artifact publication.
- Parse every workflow as YAML and run the repository documentation, lint, and
  Go test gates.
- Exercise the full suite on this pull request because it changes CI files.
