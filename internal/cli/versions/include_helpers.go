package versions

import "github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"

func normalizeAppStoreVersionInclude(value string) ([]string, error) {
	return shared.NormalizeSelection(value, appStoreVersionIncludeList(), "--include")
}

func appStoreVersionIncludeList() []string {
	return []string{
		"ageRatingDeclaration",
		"app",
		"appStoreVersionLocalizations",
		"build",
		"appStoreVersionPhasedRelease",
		"gameCenterAppVersion",
		"routingAppCoverage",
		"appStoreReviewDetail",
		"appStoreVersionSubmission",
		"appClipDefaultExperience",
		"appStoreVersionExperiments",
		"appStoreVersionExperimentsV2",
		"alternativeDistributionPackage",
	}
}
