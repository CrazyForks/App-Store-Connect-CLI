# Subscription promotional offer inline prices

## Decision

`asc subscriptions offers promotional create` remains in the existing
subscription-offers taxonomy. Its required `--prices` flag now describes the
prices Apple can create with the offer instead of IDs for promotional-offer
price resources that do not have a standalone create endpoint:

- `FREE_TRIAL`: comma-separated `TERRITORY` entries, for example `US,France`
- paid modes: comma-separated `TERRITORY:PRICE_POINT_ID` entries, for example
  `US:PRICE_POINT_ID,France:PRICE_POINT_ID`

Territories accept alpha-2, alpha-3, or exact English country names and are
normalized to App Store Connect alpha-3 IDs. Each input becomes a temporary
local linkage in `data.relationships.prices` and a matching
`subscriptionPromotionalOfferPrices` resource in top-level `included`. The
included resource links its territory and, for paid modes, its
`subscriptionPricePoint`.

## API contract

The command calls `POST /v1/subscriptionPromotionalOffers` with no query
parameters. Its request is `SubscriptionPromotionalOfferCreateRequest`; the
required primary resource has type `subscriptionPromotionalOffers`, the five
existing required attributes, and required `subscription` and `prices`
relationships. The request schema permits top-level
`SubscriptionPromotionalOfferPriceInlineCreate` resources. Each local ID used
by a price relationship exactly matches one included resource. A successful
request returns status 201 with `SubscriptionPromotionalOfferResponse`.

Apple's current official 4.4.1 OpenAPI download is byte-for-byte identical to
`docs/openapi/latest.json`. It documents create, update, and delete operations
for promotional offers but no standalone create operation for their prices.
Current Fastlane and Expo/EAS sources do not implement an alternative workflow.
Maintained App Store Connect tools that do support promotional-offer creation
use this same atomic compound POST with temporary linkages and inline territory
and price-point relationships.

The command continues to write its response through the shared TTY-aware output
path: explicit `--output` wins, data is written to stdout, and validation or API
errors are written to stderr. Missing required flags and malformed price inputs
exit with status 2 before authentication or HTTP.

## Compatibility and migration

The flag name is unchanged, but its former value shape was unusable against the
documented create contract because promotional-offer prices cannot be created
independently. A bare opaque value in a paid mode is now rejected with guidance
to use `TERRITORY:PRICE_POINT_ID`. For `FREE_TRIAL`, a bare value is interpreted
as a territory and therefore must resolve as a supported territory. Help,
examples, and generated command documentation describe the corrected shape.
Apple's schema leaves `subscriptionPricePoint` optional without a conditional
mode rule; the territory-only FREE_TRIAL shape also matches the adjacent
offer-code compound-create contract. Paid modes always require a price point.

The separate promotional-offer `update --prices` behavior is outside this
change: that endpoint updates relationships to prices belonging to an existing
offer and is not a substitute for creating the initial inline resources.

## RED-GREEN and verification

The client regression first replaces the old linkage-only mock with an exact
JSON body assertion covering temporary local IDs, normalized territory
relationships, paid price-point relationships, and top-level included
resources. CLI coverage asserts valid FREE_TRIAL and paid inputs plus malformed
mode-specific values and unknown territories before auth. The focused tests run
RED against the current linkage-only client, then GREEN after the narrow client
and parser changes.

Black-box verification uses a freshly built binary to check help, stdout,
stderr, and exit status. Live verification uses the explicitly selected auth
profile and `ASC_BYPASS_KEYCHAIN=1`: read-only calls use explicit resource IDs,
while the create-shaped probe targets a deliberately nonexistent subscription
ID so Apple rejects the request without creating a resource. The probe records
method, path, status, and error, then repeats the read-only observation needed
to show no resource appeared.

## Alternatives

Keeping pre-existing promotional-offer price IDs was rejected because there is
no supported operation that can produce those resources before the parent
offer exists. Adding separate territory and price-point flags was also rejected:
it would make multi-territory pairing ambiguous and diverge from the established
offer-code price syntax. Reusing the offer-code input grammar and inline-create
pattern gives both offer families the same mode-specific validation and request
shape.
