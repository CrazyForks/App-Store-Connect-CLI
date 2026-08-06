# Remove the broken app search-keywords setter

## Placement and command shape

`asc apps search-keywords list` remains the low-level app keyword read surface.
The sibling `set` leaf is removed because Apple has no app-level keyword write
operation and the command has no version or locale inputs with which to perform
the supported metadata update.

Keyword text remains writable through the existing version-localization
workflows:

- `asc metadata keywords apply --app "APP_ID" --version "1.2.3" --dir
  "./metadata" --confirm` for repository-backed metadata;
- `asc metadata keywords push --version-id "VERSION_ID" --input
  "./keywords.json"` for direct locale-keyed input; or
- `asc localizations update --version "VERSION_ID" --locale "en-US"
  --keywords "kw1,kw2"` for a single localization.

The broken `AppsSearchKeywordsSetCommand`, its flags and examples, the
`SetAppSearchKeywords` client method, and its synthetic PATCH/204 mock are
deleted. Shared keyword response helpers and version-localization relationship
commands remain because supported localization and custom-product-page flows
use them.

## API contract and maintained implementations

Apple's current OpenAPI exposes only `GET` at
`/v1/apps/{id}/relationships/searchKeywords`; it accepts `limit` and returns
`AppSearchKeywordsLinkagesResponse`. The related
`GET /v1/apps/{id}/searchKeywords` accepts `filter[platform]`,
`filter[locale]`, `fields[appKeywords]`, and `limit`, and returns
`AppKeywordsResponse`. There is no app-level create, replace, update, or delete
operation.

The supported text-keyword contract is
`PATCH /v1/appStoreVersionLocalizations/{id}` with
`AppStoreVersionLocalizationUpdateRequest.data.attributes.keywords`. Fastlane
Deliver and Expo EAS Metadata both use this version-localization attribute.
Expo scopes metadata by App Store version and locale; Fastlane updates each
`appStoreVersionLocalization`. Neither uses the app relationship. Codemagic's
maintained publishing workflow likewise models keywords as localized release
metadata. No inspected maintained implementation replaces app-level
`searchKeywords`.

Localization-level `searchKeywords` relationship POST and DELETE operations do
exist, but accept opaque `appKeywords` IDs rather than raw keyword text. Those
supported commands are outside this defect and remain unchanged.

## History and live behavior

Commit `56e162bb7ea0e2a9e7c3471ea2ffedf163411c1f` introduced `set` in PR #346
for issue #317. The OpenAPI snapshot at that commit already exposed only GET
for the app relationship. The PR's live smoke exercised `list`, not `set`; its
only set evidence was a mock that accepted PATCH and returned 204. No issue,
review, or later history records a successful app-level replacement.

The unmodified built CLI sent
`PATCH /v1/apps/0000000000/relationships/searchKeywords` with deliberately
nonexistent input. Apple returned HTTP 403 and reported that `REPLACE` is not
allowed; only `GET_RELATED` and `GET_RELATIONSHIP` are allowed. A read-only GET
confirmed that app ID does not exist, proving the probe changed no resource. A
read-only call to the disposable app's related endpoint with explicit locale
and platform returned HTTP 200, confirming the neighboring read path.

## Compatibility and verification

The normal stable-command deprecation policy would preserve a compatibility
stub. For this change, the user explicitly authorizes direct removal because
the surface never had a supported contract or evidence of working. Keeping a
fail-closed stub would preserve dead surface area, while transparently
rerouting would guess a version and locale that the old flags never supplied.

No bespoke test is added merely to assert absence. The misleading client mock
and CLI validation cases are deleted. Focused verification exercises the real
app keyword read client/command and version-localization metadata keyword
workflows, followed by generated command documentation, built-binary help and
read-only live checks, and the full formatting, documentation, lint, and test
gate.
