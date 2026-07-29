package urlsanitize

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const (
	signatureSentinel = "asc-red-sentinel-urlsanitize-sig-1c7f04"
	pathSentinel      = "asc-red-sentinel-urlsanitize-path-90ab6d"
	userinfoSentinel  = "asc-red-sentinel-urlsanitize-user-42de81"
)

func TestRedactURLForErrorKeepsPathAndDropsCredentials(t *testing.T) {
	raw := "https://" + userinfoSentinel + ":pw@upload.example.com/v1/assets/chunk-1" +
		"?X-Amz-Signature=" + signatureSentinel + "#" + signatureSentinel

	got := RedactURLForError(raw)

	if got != "https://upload.example.com/v1/assets/chunk-1" {
		t.Fatalf("RedactURLForError() = %q", got)
	}
	for _, sentinel := range []string{signatureSentinel, userinfoSentinel} {
		if strings.Contains(got, sentinel) {
			t.Fatalf("RedactURLForError() leaked %q: %q", sentinel, got)
		}
	}
}

func TestRedactURLHostForErrorDropsPath(t *testing.T) {
	raw := "https://hooks.slack.com/services/T00000000/B00000000/" + pathSentinel

	got := RedactURLHostForError(raw)

	if got != "https://hooks.slack.com" {
		t.Fatalf("RedactURLHostForError() = %q", got)
	}
}

func TestRedactHelpersFallBackToPlaceholder(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty", raw: ""},
		{name: "whitespace", raw: "   "},
		{name: "unparseable", raw: "https://exa mple.com/\x7f"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RedactURLForError(test.raw); got != RedactedPlaceholder {
				t.Fatalf("RedactURLForError(%q) = %q, want %q", test.raw, got, RedactedPlaceholder)
			}
			if got := RedactURLHostForError(test.raw); got != RedactedPlaceholder {
				t.Fatalf("RedactURLHostForError(%q) = %q, want %q", test.raw, got, RedactedPlaceholder)
			}
		})
	}
}

func TestRedactURLHostForErrorRejectsHostlessURLs(t *testing.T) {
	if got := RedactURLHostForError("/services/T00000000/" + pathSentinel); got != RedactedPlaceholder {
		t.Fatalf("RedactURLHostForError() = %q, want %q", got, RedactedPlaceholder)
	}
}

func TestNewTransportErrorClassifiesAndStaysInspectable(t *testing.T) {
	cause := errors.New("connect: connection refused")
	err := NewTransportError("upload request", "https://upload.example.com/chunk-1", cause)

	if err.Error() != "upload request failed for https://upload.example.com/chunk-1" {
		t.Fatalf("Error() = %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is lost the cause: %v", err)
	}

	timeout := NewTransportError("upload request", "https://upload.example.com/chunk-1", context.DeadlineExceeded)
	if !strings.Contains(timeout.Error(), "(timeout)") {
		t.Fatalf("timeout classification missing from %q", timeout.Error())
	}
	canceled := NewTransportError("upload request", "https://upload.example.com/chunk-1", context.Canceled)
	if !strings.Contains(canceled.Error(), "(canceled)") {
		t.Fatalf("cancellation classification missing from %q", canceled.Error())
	}
}

func TestClassifyTransportFailureIgnoresUnknownCauses(t *testing.T) {
	if got := ClassifyTransportFailure(nil); got != "" {
		t.Fatalf("ClassifyTransportFailure(nil) = %q, want empty", got)
	}
	if got := ClassifyTransportFailure(errors.New("tls: handshake failure")); got != "" {
		t.Fatalf("ClassifyTransportFailure(tls) = %q, want empty", got)
	}
}
