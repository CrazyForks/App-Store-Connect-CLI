# Category subcategories pagination contract

## Placement and command shape

`asc categories subcategories` remains the stable parent-scoped collection
command under the existing `asc categories` taxonomy. No registry or command
name changes are required. Current help exposes these flags:

- `--category-id`
- `--limit`
- `--next`
- `--paginate`
- `--output`
- `--pretty`

The valid invocation shapes remain category-scoped listing, next-page
continuation, and aggregation from either starting point:

```bash
asc categories subcategories --category-id "GAMES"
asc categories subcategories --category-id "GAMES" --limit 25
asc categories subcategories --next "<links.next>"
asc categories subcategories --next "<links.next>" --paginate
```

## API and output contract

The command issues `GET /v1/appCategories/{id}/subcategories`, whose OpenAPI
operation is `appCategories_subcategories_getToManyRelated`. The endpoint
accepts `fields[appCategories]` and `limit` query parameters; this command
currently exposes only `limit`. A successful request returns
`AppCategoriesWithoutIncludesResponse`, including an opaque `links.next` URL
for continuation.

When `--next` is non-empty, its URL replaces the complete parent-scoped path
and query string. Explicit `--category-id` and `--limit` values therefore
cannot also shape that request and must be rejected instead of silently
ignored. The rejection is based on flag presence, so explicit empty or zero
values are conflicts too. `--paginate`, `--output`, and `--pretty` remain valid
with `--next` because they control continuation or local rendering rather than
the first request URL.

Validation runs before authentication or HTTP in this order:

1. Validate a non-empty `--next` URL.
2. Reject explicitly visited `--category-id` or `--limit` flags with `--next`.
3. Validate the standalone `--limit` range.
4. Require either a category ID or a next URL.

This preserves the invalid-next error when an invalid URL is paired with a
conflicting flag, while a valid next URL plus an explicit conflict returns a
usage error with exit code 2. Data continues to use stdout, and validation
diagnostics continue to use stderr.

## Compatibility and failure modes

Next-only, category-only, category-plus-limit, pagination, and explicit output
behavior remain unchanged. Omitting `--limit` retains its existing default.
The only compatibility change is that combinations whose category or limit
was previously accepted and discarded now fail closed with a usage error.
Command help is unchanged, so generated command documentation should not
change.

A malformed or non-App-Store-Connect next URL fails before conflict checking.
A valid next URL plus `--category-id`, `--category-id ""`, `--limit 0`, or an
out-of-range explicit limit fails as a cursor conflict. An out-of-range limit
without `--next` retains its existing range error. API and later-page failures
retain the current command context.

## Verification

RED command tests will prove that explicit parent and limit flags currently
reach the next URL without an error. GREEN coverage will assert both flag
orders, explicit empty and zero values, out-of-range limit precedence, invalid
next precedence, and validation before client creation. HTTP-backed controls
will preserve the exact next-only URL, category-only path and limit query, and
the existing paginate-from-next behavior.

A built binary will verify stdout, stderr, and exit codes for the conflict,
invalid-next, and standalone invalid-limit cases with
`ASC_BYPASS_KEYCHAIN=1`. Focused command tests precede the repository format,
documentation, lint, and full test gates. A live API call is unnecessary for
this pre-request contract because the OpenAPI operation and local HTTP request
shape fully determine the behavior.

One alternative was to parse the parent ID from the next URL and compare it
with `--category-id`. That couples the stable CLI to the internal structure of
an opaque continuation URL and still leaves `--limit` ambiguous. Another was
to rewrite the next URL with explicitly supplied values, which could invalidate
the server-issued cursor. Rejecting the two request-shaping flags is narrower
and consistent with the URL's authoritative role.
