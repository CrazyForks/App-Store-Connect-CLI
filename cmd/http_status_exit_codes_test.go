package cmd

import (
	"fmt"
	"net/http"
	"testing"

	webcore "github.com/rudrankriyam/App-Store-Connect-CLI/internal/web"
)

func TestExitCodeFromError_WebAPIStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected int
	}{
		{name: "not found", status: http.StatusNotFound, expected: ExitNotFound},
		{name: "conflict", status: http.StatusConflict, expected: ExitConflict},
		{name: "server error", status: http.StatusInternalServerError, expected: ExitHTTPInternalServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("web command failed: %w", &webcore.APIError{Status: tt.status})
			if got := ExitCodeFromError(err); got != tt.expected {
				t.Fatalf("ExitCodeFromError() = %d, want %d", got, tt.expected)
			}
		})
	}
}
