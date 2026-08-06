package asc

import (
	"context"
	"net/http"
	"testing"
)

func TestGetAppStoreReviewAttachmentsForReviewDetail_UsesNextURLWithoutReviewDetail(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/appStoreReviewDetails/detail-1/appStoreReviewAttachments?cursor=abc"
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.String() != next {
			t.Fatalf("expected URL %q, got %q", next, req.URL.String())
		}
		assertAuthorized(t, req)
	}, jsonResponse(http.StatusOK, `{"data":[],"links":{"self":"`+next+`"}}`))

	if _, err := client.GetAppStoreReviewAttachmentsForReviewDetail(
		context.Background(),
		"",
		WithAppStoreReviewAttachmentsNextURL(next),
	); err != nil {
		t.Fatalf("GetAppStoreReviewAttachmentsForReviewDetail() error: %v", err)
	}
}
