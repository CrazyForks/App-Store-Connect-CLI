package asc

// RedactedValuePlaceholder marks a secret that was withheld from output. It is
// deliberately not a valid App Store Connect value so a redacted payload can
// never be mistaken for a usable credential.
const RedactedValuePlaceholder = "(redacted)"

// RedactSecret returns the placeholder for any non-empty secret and leaves the
// empty value untouched so absent fields stay absent in rendered output. The
// API permits an unconstrained string, so even a whitespace-only value is
// treated as a credential.
func RedactSecret(value string) string {
	if value == "" {
		return value
	}
	return RedactedValuePlaceholder
}

// RedactAppStoreReviewDetailAttributes returns a presentation-safe copy of App
// Store review detail attributes.
func RedactAppStoreReviewDetailAttributes(attrs AppStoreReviewDetailAttributes) AppStoreReviewDetailAttributes {
	attrs.DemoAccountPassword = RedactSecret(attrs.DemoAccountPassword)
	return attrs
}

// RedactBetaAppReviewDetailAttributes returns a presentation-safe copy of
// TestFlight beta app review detail attributes.
func RedactBetaAppReviewDetailAttributes(attrs BetaAppReviewDetailAttributes) BetaAppReviewDetailAttributes {
	attrs.DemoAccountPassword = RedactSecret(attrs.DemoAccountPassword)
	return attrs
}

// RedactAppStoreReviewDetailResponse returns a presentation-safe copy of a
// single App Store review detail response. The argument is left unmodified so
// callers keep the real password for requests, comparison, and validation.
func RedactAppStoreReviewDetailResponse(resp *AppStoreReviewDetailResponse) *AppStoreReviewDetailResponse {
	return redactSingleResponse(resp, RedactAppStoreReviewDetailAttributes)
}

// RedactBetaAppReviewDetailResponse returns a presentation-safe copy of a
// single TestFlight beta app review detail response.
func RedactBetaAppReviewDetailResponse(resp *BetaAppReviewDetailResponse) *BetaAppReviewDetailResponse {
	return redactSingleResponse(resp, RedactBetaAppReviewDetailAttributes)
}

// RedactBetaAppReviewDetailsResponse returns a presentation-safe copy of a
// TestFlight beta app review details list response.
func RedactBetaAppReviewDetailsResponse(resp *BetaAppReviewDetailsResponse) *BetaAppReviewDetailsResponse {
	return redactListResponse(resp, RedactBetaAppReviewDetailAttributes)
}

func redactSingleResponse[T any](resp *SingleResponse[T], redact func(T) T) *SingleResponse[T] {
	if resp == nil {
		return nil
	}
	safe := *resp
	safe.Data.Attributes = redact(safe.Data.Attributes)
	return &safe
}

func redactListResponse[T any](resp *Response[T], redact func(T) T) *Response[T] {
	if resp == nil {
		return nil
	}
	safe := *resp
	if len(resp.Data) > 0 {
		safe.Data = make([]Resource[T], len(resp.Data))
		copy(safe.Data, resp.Data)
		for i := range safe.Data {
			safe.Data[i].Attributes = redact(safe.Data[i].Attributes)
		}
	}
	return &safe
}
