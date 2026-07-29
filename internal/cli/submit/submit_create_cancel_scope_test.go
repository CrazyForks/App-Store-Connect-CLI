package submit

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// TestPrepareReviewSubmissionForCreateDoesNotCancelSubmissionForAnotherVersion
// proves that preparation never withdraws a review submission holding another
// version's review items.
func TestPrepareReviewSubmissionForCreateDoesNotCancelSubmissionForAnotherVersion(t *testing.T) {
	requests := make([]string, 0, 3)
	client := newSubmitTestClient(t, submitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.RequestURI())

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/reviewSubmissions":
			return submitJSONResponse(http.StatusOK, `{
				"data": [{
					"type": "reviewSubmissions",
					"id": "other-version-submission",
					"attributes": {"state": "READY_FOR_REVIEW", "platform": "IOS"},
					"relationships": {
						"appStoreVersionForReview": {
							"data": {"type": "appStoreVersions", "id": "version-2"}
						}
					}
				}]
			}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/other-version-submission/items":
			return submitJSONResponse(http.StatusOK, `{
				"data": [{
					"type": "reviewSubmissionItems",
					"id": "other-version-item",
					"relationships": {
						"appStoreVersion": {
							"data": {"type": "appStoreVersions", "id": "version-2"}
						}
					}
				}]
			}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.RequestURI())
		}
	}))

	stderr := captureSubmitStderr(t, func() {
		got := prepareReviewSubmissionForCreate(context.Background(), client, "app-1", "IOS", "version-1", nil)
		if got.reuseSubmissionID != "" {
			t.Fatalf("expected no reusable submission, got %#v", got)
		}
		if got.canceledSubmissionIDs != nil {
			t.Fatalf("expected no canceled submissions, got %#v", got.canceledSubmissionIDs)
		}
	})

	wantRequests := []string{
		"GET /v1/apps/app-1/reviewSubmissions?filter%5Bplatform%5D=IOS&filter%5Bstate%5D=READY_FOR_REVIEW&include=appStoreVersionForReview&limit=200",
		"GET /v1/reviewSubmissions/other-version-submission/items?limit=200",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("unexpected requests: got %v want %v", requests, wantRequests)
	}
	if !strings.Contains(stderr, "Skipped stale review submission other-version-submission") {
		t.Fatalf("expected skip diagnostic naming the submission, got %q", stderr)
	}
	if !strings.Contains(stderr, "asc submit cancel") {
		t.Fatalf("expected skip diagnostic to point at explicit cancellation, got %q", stderr)
	}
}

// TestPrepareReviewSubmissionForCreateDoesNotCancelUnprovenSubmission proves
// that a failed item lookup blocks cancellation instead of falling back to it.
func TestPrepareReviewSubmissionForCreateDoesNotCancelUnprovenSubmission(t *testing.T) {
	requests := make([]string, 0, 3)
	client := newSubmitTestClient(t, submitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.RequestURI())

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/reviewSubmissions":
			return submitJSONResponse(http.StatusOK, `{
				"data": [{
					"type": "reviewSubmissions",
					"id": "unproven-submission",
					"attributes": {"state": "READY_FOR_REVIEW", "platform": "IOS"}
				}]
			}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/unproven-submission/items":
			return submitJSONResponse(http.StatusInternalServerError, `{"errors":[{"status":"500","title":"Server error"}]}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.RequestURI())
		}
	}))

	stderr := captureSubmitStderr(t, func() {
		got := prepareReviewSubmissionForCreate(context.Background(), client, "app-1", "IOS", "version-1", nil)
		if got.canceledSubmissionIDs != nil {
			t.Fatalf("expected no canceled submissions, got %#v", got.canceledSubmissionIDs)
		}
	})

	for _, request := range requests {
		if strings.HasPrefix(request, http.MethodPatch) {
			t.Fatalf("expected no cancellation request, got %v", requests)
		}
	}
	if !strings.Contains(stderr, "Skipped stale review submission unproven-submission") {
		t.Fatalf("expected skip diagnostic naming the submission, got %q", stderr)
	}
}

// TestPrepareReviewSubmissionForCreateCancelsSubmissionProvenEmpty keeps the
// working path: a submission with no review items is safe to withdraw.
func TestPrepareReviewSubmissionForCreateCancelsSubmissionProvenEmpty(t *testing.T) {
	requests := make([]string, 0, 3)
	client := newSubmitTestClient(t, submitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.RequestURI())

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/reviewSubmissions":
			return submitJSONResponse(http.StatusOK, `{
				"data": [{
					"type": "reviewSubmissions",
					"id": "empty-submission",
					"attributes": {"state": "READY_FOR_REVIEW", "platform": "IOS"}
				}]
			}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/empty-submission/items":
			return submitJSONResponse(http.StatusOK, `{"data":[],"links":{}}`)
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/reviewSubmissions/empty-submission":
			return submitJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"empty-submission"}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.RequestURI())
		}
	}))

	captureSubmitStderr(t, func() {
		got := prepareReviewSubmissionForCreate(context.Background(), client, "app-1", "IOS", "version-1", nil)
		if _, ok := got.canceledSubmissionIDs["empty-submission"]; !ok {
			t.Fatalf("expected empty submission to be canceled, got %#v", got.canceledSubmissionIDs)
		}
	})

	wantRequests := []string{
		"GET /v1/apps/app-1/reviewSubmissions?filter%5Bplatform%5D=IOS&filter%5Bstate%5D=READY_FOR_REVIEW&include=appStoreVersionForReview&limit=200",
		"GET /v1/reviewSubmissions/empty-submission/items?limit=200",
		"PATCH /v1/reviewSubmissions/empty-submission",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("unexpected requests: got %v want %v", requests, wantRequests)
	}
}
