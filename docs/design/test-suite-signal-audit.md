# Test suite signal audit

## Scope

This change removes tests that do not protect shipped behavior. It does not
change the CLI command hierarchy, flags, output, exit codes, API requests, or
authentication behavior.

The audit covered every Go test file and top-level test, exact duplicate test
bodies, skipped tests, the largest and slowest packages, command-constructor
smoke tests, and packages outside the dependency graph of the shipped CLI and
Studio binaries.

## Findings

Two groups are safe to remove:

1. Twenty `Test*CommandConstructors` tests only call constructors and check
   that concrete `*ffcli.Command` values are non-nil, sometimes also checking
   that a name and subcommand list are non-empty. The registry test constructs
   every registered command tree, so these tests duplicate construction while
   asserting less than the registry and command-level behavior tests.
2. `internal/iris` is absent from the dependency graph of the CLI and Studio
   binaries. It is an obsolete predecessor of `internal/web`; no production Go
   file imports it. Its test file also contains the only exact duplicate test
   bodies found by the audit, duplicated in the active `internal/web` package.
   Removing only the tests would leave an untested dead implementation, so the
   obsolete package is removed as one unit.

Conditional platform and integration skips remain because they document
specific unavailable capabilities. Command-shape tests that assert public
names or exact subcommand surfaces remain. Validation, HTTP, output, auth,
artifact, and regression tests remain.

## Verification design

There is no RED test for deleting non-behavioral coverage: adding a test that
asserts these tests or files are absent would recreate the same maintenance
problem. The pre-change characterization is a green full suite plus the
shipped dependency graph, which contains `internal/web` and excludes
`internal/iris`.

After removal, run focused tests for every affected active command package,
then the repository format, documentation, lint, build, and full test gates.
Verify the built binary can construct help and perform one read-only,
authenticated App Store Connect request. No live mutation is needed.

## Alternatives

- Keeping the tests preserves nominal coverage percentages but not useful
  behavioral protection.
- Deleting only `internal/iris/auth_test.go` reduces duplication while leaving
  dead production code behind and is therefore less coherent than deleting the
  unreachable package.
