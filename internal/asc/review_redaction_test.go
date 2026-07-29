package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

const reviewRedactionSentinel = "asc-red-sentinel-asc-demo-pw-2ad57e"

func TestRedactSecretPreservesEmptyValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty stays empty", value: "", want: ""},
		{name: "whitespace stays whitespace", value: "   ", want: "   "},
		{name: "secret becomes placeholder", value: reviewRedactionSentinel, want: RedactedValuePlaceholder},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RedactSecret(test.value); got != test.want {
				t.Fatalf("RedactSecret(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestRedactAppStoreReviewDetailResponseLeavesOriginalIntact(t *testing.T) {
	original := &AppStoreReviewDetailResponse{}
	original.Data.ID = "detail-1"
	original.Data.Attributes = AppStoreReviewDetailAttributes{
		DemoAccountName:     "reviewer@example.com",
		DemoAccountPassword: reviewRedactionSentinel,
		Notes:               "Reviewer notes",
	}

	safe := RedactAppStoreReviewDetailResponse(original)

	if original.Data.Attributes.DemoAccountPassword != reviewRedactionSentinel {
		t.Fatalf("original password was mutated to %q", original.Data.Attributes.DemoAccountPassword)
	}
	if safe.Data.Attributes.DemoAccountPassword != RedactedValuePlaceholder {
		t.Fatalf("redacted password = %q, want %q", safe.Data.Attributes.DemoAccountPassword, RedactedValuePlaceholder)
	}
	if safe.Data.ID != "detail-1" || safe.Data.Attributes.DemoAccountName != "reviewer@example.com" {
		t.Fatalf("redaction dropped non-sensitive fields: %+v", safe.Data)
	}

	encoded, err := json.Marshal(safe)
	if err != nil {
		t.Fatalf("marshal redacted response: %v", err)
	}
	if strings.Contains(string(encoded), reviewRedactionSentinel) {
		t.Fatalf("serialized redacted response leaked the sentinel: %s", encoded)
	}
}

func TestRedactBetaAppReviewDetailsResponseCopiesEveryResource(t *testing.T) {
	original := &BetaAppReviewDetailsResponse{
		Data: []Resource[BetaAppReviewDetailAttributes]{
			{ID: "detail-1", Attributes: BetaAppReviewDetailAttributes{DemoAccountPassword: reviewRedactionSentinel}},
			{ID: "detail-2", Attributes: BetaAppReviewDetailAttributes{DemoAccountName: "reviewer@example.com"}},
		},
	}

	safe := RedactBetaAppReviewDetailsResponse(original)

	if original.Data[0].Attributes.DemoAccountPassword != reviewRedactionSentinel {
		t.Fatal("redaction mutated the original list response")
	}
	if safe.Data[0].Attributes.DemoAccountPassword != RedactedValuePlaceholder {
		t.Fatalf("first resource password = %q, want placeholder", safe.Data[0].Attributes.DemoAccountPassword)
	}
	if safe.Data[1].Attributes.DemoAccountPassword != "" {
		t.Fatalf("empty password became %q, want empty", safe.Data[1].Attributes.DemoAccountPassword)
	}
	if safe.Data[1].Attributes.DemoAccountName != "reviewer@example.com" {
		t.Fatalf("redaction dropped non-sensitive fields: %+v", safe.Data[1])
	}
}

func TestRedactResponseHelpersHandleNil(t *testing.T) {
	if got := RedactAppStoreReviewDetailResponse(nil); got != nil {
		t.Fatalf("RedactAppStoreReviewDetailResponse(nil) = %v, want nil", got)
	}
	if got := RedactBetaAppReviewDetailResponse(nil); got != nil {
		t.Fatalf("RedactBetaAppReviewDetailResponse(nil) = %v, want nil", got)
	}
	if got := RedactBetaAppReviewDetailsResponse(nil); got != nil {
		t.Fatalf("RedactBetaAppReviewDetailsResponse(nil) = %v, want nil", got)
	}
}
