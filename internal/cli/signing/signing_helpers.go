package signing

import (
	"fmt"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

var signingPlatformValues = map[string]asc.Platform{
	"IOS":       asc.PlatformIOS,
	"MAC_OS":    asc.PlatformMacOS,
	"TV_OS":     asc.PlatformTVOS,
	"VISION_OS": asc.PlatformVisionOS,
}

func normalizePlatform(value string) (asc.Platform, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("--platform is required")
	}
	platform, ok := signingPlatformValues[normalized]
	if !ok {
		return "", fmt.Errorf("--platform must be one of: %s", strings.Join(signingPlatformList(), ", "))
	}
	return platform, nil
}

func normalizePlatforms(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToUpper(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		if _, ok := signingPlatformValues[trimmed]; !ok {
			return nil, fmt.Errorf("--platform must be one of: %s", strings.Join(signingPlatformList(), ", "))
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	return normalized, nil
}

func signingPlatformList() []string {
	return []string{"IOS", "MAC_OS", "TV_OS", "VISION_OS"}
}
