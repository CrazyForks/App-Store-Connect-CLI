package asc

import (
	"strings"
	"testing"
)

func TestAPIErrorError_SanitizesControlCharacters(t *testing.T) {
	err := &APIError{
		Title:  "Bad\x1b[31m",
		Detail: "Detail\x07",
		Code:   "CODE\x1b",
	}

	message := err.Error()
	if strings.ContainsAny(message, "\x1b\x07") {
		t.Fatalf("expected control characters to be stripped, got %q", message)
	}
	if !strings.Contains(message, "Bad") || !strings.Contains(message, "Detail") {
		t.Fatalf("expected title and detail in message, got %q", message)
	}
}
