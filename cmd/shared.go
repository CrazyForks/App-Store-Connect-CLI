package cmd

import (
	"context"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// Deprecated: use CleanupTempPrivateKeys to remove all tracked temp keys.
func CleanupTempPrivateKey() {
	shared.CleanupTempPrivateKey()
}

func CleanupTempPrivateKeys() {
	shared.CleanupTempPrivateKeys()
}

func Bold(s string) string {
	return shared.Bold(s)
}

func DefaultUsageFunc(c *ffcli.Command) string {
	return shared.DefaultUsageFunc(c)
}

func getASCClient() (*asc.Client, error) {
	return shared.GetASCClient()
}

func resolveProfileName() string {
	return shared.ResolveProfileName()
}

func resolvePrivateKeyPath() (string, error) {
	return shared.ResolvePrivateKeyPath()
}

func printOutput(data interface{}, format string, pretty bool) error {
	return shared.PrintOutput(data, format, pretty)
}

func normalizeDate(value, flagName string) (string, error) {
	return shared.NormalizeDate(value, flagName)
}

func isAppAvailabilityMissing(err error) bool {
	return shared.IsAppAvailabilityMissing(err)
}

func resolveAppID(appID string) string {
	return shared.ResolveAppID(appID)
}

func contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return shared.ContextWithTimeout(ctx)
}

func contextWithUploadTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return shared.ContextWithUploadTimeout(ctx)
}

func splitCSV(value string) []string {
	return shared.SplitCSV(value)
}

func splitCSVUpper(value string) []string {
	return shared.SplitCSVUpper(value)
}

func validateNextURL(next string) error {
	return shared.ValidateNextURL(next)
}

func validateSort(value string, allowed ...string) error {
	return shared.ValidateSort(value, allowed...)
}
