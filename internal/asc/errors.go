package asc

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrBadRequest   = errors.New("bad request")
)

// APIError represents a parsed App Store Connect error response.
type APIError struct {
	Code   string
	Title  string
	Detail string
}

func (e *APIError) Error() string {
	title := strings.TrimSpace(sanitizeTerminal(e.Title))
	detail := strings.TrimSpace(sanitizeTerminal(e.Detail))
	code := strings.TrimSpace(sanitizeTerminal(e.Code))
	switch {
	case title != "" && detail != "":
		return fmt.Sprintf("%s: %s", title, detail)
	case title != "":
		return title
	case detail != "":
		return detail
	case code != "":
		return code
	default:
		return "API error"
	}
}

func (e *APIError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return strings.EqualFold(e.Code, "NOT_FOUND")
	case ErrUnauthorized:
		return strings.EqualFold(e.Code, "UNAUTHORIZED")
	case ErrForbidden:
		return strings.EqualFold(e.Code, "FORBIDDEN")
	case ErrBadRequest:
		return strings.EqualFold(e.Code, "BAD_REQUEST")
	default:
		return false
	}
}
