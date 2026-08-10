# Analytics request timeout scoping

## Placement and invocation

This change stays inside the existing `analytics view` and
`analytics download` commands. It does not add a command, flag, registry entry,
or environment variable. The current invocation shapes remain:

```sh
asc analytics view --request-id "REQUEST_ID" --paginate
asc analytics view --request-id "REQUEST_ID" --include-segments
asc analytics download --request-id "REQUEST_ID" --instance-id "INSTANCE_ID"
asc analytics download --request-id "REQUEST_ID" --instance-id "INSTANCE_ID" --segment-id "SEGMENT_ID"
```

`ASC_TIMEOUT` and `ASC_TIMEOUT_SECONDS` continue to define the standard timeout
for one outbound request. Each analytics page, per-report instance lookup,
per-instance segment lookup, and final report transfer receives a fresh child
timeout derived from the command's parent context.

## API contract

The checked-in OpenAPI snapshot documents these read-only operations:

- `GET /v1/analyticsReportRequests/{id}/reports`, with an optional `limit` and
  an `AnalyticsReportsResponse` collection.
- `GET /v1/analyticsReports/{id}/instances`, with optional granularity,
  processing-date, and limit filters and an `AnalyticsReportInstancesResponse`
  collection.
- `GET /v1/analyticsReportInstances/{id}/segments`, with an optional limit and
  an `AnalyticsReportSegmentsResponse` collection.

Collection responses can carry `links.next`, so each page is an independent
HTTP request. An analytics report segment exposes a URI-valued `url` attribute.
`analytics download` retrieves that presigned URL without App Store Connect
authorization and streams its body to the selected output path. The endpoint
paths, methods, queries, response decoding, pagination, and download URL
validation remain unchanged.

## Output, exit, and compatibility contract

There is no public command-shape or output-schema change. `analytics view`
continues to print the selected report, instance, and optional segment data in
the requested JSON, table, or Markdown format. `analytics download` continues
to write the compressed report, optionally decompress it, and print metadata
about the artifact. Data remains on stdout, errors on stderr, validation exits
remain unchanged, and no migration or deprecation is required.

The only behavior change is timeout scope. A slow or failed page still stops the
command after its own configured request timeout, but elapsed time spent on an
earlier successful request no longer consumes the deadline of an independent
later request. A caller cancellation or earlier parent deadline still stops the
active request and prevents subsequent work because every child context derives
from that parent.

## RED-GREEN and verification

Command-level regression coverage uses an isolated authenticated client and a
deterministic local HTTP transport. It records each request deadline across
report pagination, instance and segment discovery, and final download. The RED
must show that current later requests inherit the first absolute deadline. GREEN
requires a later deadline for every independently started request, while the
download body confirms its request context remains live for the whole stream.

Existing selection and structured-output tests remain the compatibility guard.
Focused cancellation coverage verifies that canceling the command parent still
cancels the active child. A task-specific `/tmp` binary will run against a local
mock service to verify request ordering, output streams, artifact contents, and
exit status. Because the defect is local context orchestration and the endpoint
contract is already represented by deterministic HTTP fixtures, no live Apple
request or mutation is required.

Repository verification is:

```sh
make build
make format
make check-docs
make lint
ASC_BYPASS_KEYCHAIN=1 make test
```

## Edge cases and failure modes

- Repeated pagination URLs retain their existing fail-closed detection.
- A parent cancellation or shorter parent deadline always wins over a newly
  resolved request timeout.
- The final download timeout covers both response establishment and body copy;
  it is released before optional decompression and output rendering.
- A failed request, body read, or artifact write retains the existing wrapped
  command error and does not proceed to later discovery or output steps.
- Report, instance, and segment filtering and first-match selection remain
  sequential and unchanged.

## Alternatives

Keeping one command-wide deadline is simpler, but it makes `ASC_TIMEOUT` depend
on how many successful pages precede a request and can expire a later request
before it receives its own request budget. Raising the global timeout or adding
a command flag would expose a workaround rather than correct the shared-deadline
semantics. Setting an `http.Client.Timeout` would spread policy into the client,
would not express the existing parent-cancellation contract as directly, and
would be broader than these composite commands. Fresh child contexts at each
existing request boundary are the smallest compatible fix.
