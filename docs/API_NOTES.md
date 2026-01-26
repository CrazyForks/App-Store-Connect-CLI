# API Notes

Quirks and tips for specific App Store Connect API endpoints.

## Analytics & Sales Reports

- Date formats vary by frequency:
  - DAILY/WEEKLY: `YYYY-MM-DD`
  - MONTHLY: `YYYY-MM`
  - YEARLY: `YYYY`
- Vendor number comes from Sales and Trends → Reports URL (`vendorNumber=...`)
- Use `--paginate` with `asc analytics get --date` to avoid missing instances on later pages
- Long analytics runs may require raising `ASC_TIMEOUT`

## Finance Reports

Finance reports use Apple fiscal months (`YYYY-MM`), not calendar months.

**API Report Types (mapping to App Store Connect UI):**

| API `--report-type` | UI Option                               | `--region` Code(s)      |
|---------------------|-----------------------------------------|-------------------------|
| `FINANCIAL`         | All Countries or Regions (Single File)  | `ZZ` (consolidated)     |
| `FINANCIAL`         | All Countries or Regions (Multiple Files) | `US`, `EU`, `JP`, etc. |
| `FINANCE_DETAIL`    | All Countries or Regions (Detailed)     | `Z1` (required)         |
| Not available       | Transaction Tax (Single File)           | N/A                     |

**Important:**
- `FINANCE_DETAIL` reports require region code `Z1` (the only valid region for detailed reports)
- Transaction Tax reports are NOT available via API; download manually from App Store Connect
- Region codes reference: https://developer.apple.com/help/app-store-connect/reference/financial-report-regions-and-currencies/
- Use `asc finance regions` to see all available region codes

## Sandbox Testers

- Required fields: email, first/last name, password + confirm, secret question/answer, birth date, territory
- Password must include uppercase, lowercase, and a number (8+ chars)
- Territory uses 3-letter App Store territory codes (e.g., `USA`, `JPN`)
- List/get use the v2 API; create/delete use v1 endpoints (may be unavailable on some accounts)
- Update/clear-history use the v2 API

## Devices

- No DELETE endpoint; devices can only be enabled/disabled via PATCH.
- Registration requires a UDID (iOS) or Hardware UUID (macOS).
- Device management UI lives in the Apple Developer portal, not App Store Connect.
- Device reset is limited to once per membership year; disabling does not free slots.
