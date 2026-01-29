package cmd

import "github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"

// ReportedError marks an error as already reported to the user.
// The main entrypoint should exit non-zero without duplicating output.
type ReportedError = shared.ReportedError

// NewReportedError wraps an error that has already been printed.
func NewReportedError(err error) error {
	return shared.NewReportedError(err)
}
