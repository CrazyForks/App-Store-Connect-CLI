package asc

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func printGameCenterAchievementsTable(resp *GameCenterAchievementsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tVendor ID\tPoints\tShow Before Earned\tRepeatable\tArchived")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%t\t%t\t%t\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.VendorIdentifier,
			item.Attributes.Points,
			item.Attributes.ShowBeforeEarned,
			item.Attributes.Repeatable,
			item.Attributes.Archived,
		)
	}
	return w.Flush()
}

func printGameCenterAchievementsMarkdown(resp *GameCenterAchievementsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Vendor ID | Points | Show Before Earned | Repeatable | Archived |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %d | %t | %t | %t |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(item.Attributes.VendorIdentifier),
			item.Attributes.Points,
			item.Attributes.ShowBeforeEarned,
			item.Attributes.Repeatable,
			item.Attributes.Archived,
		)
	}
	return nil
}

func printGameCenterAchievementDeleteResultTable(result *GameCenterAchievementDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterAchievementDeleteResultMarkdown(result *GameCenterAchievementDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardsTable(resp *GameCenterLeaderboardsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tVendor ID\tFormatter\tSort\tSubmission Type\tArchived")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%t\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.VendorIdentifier,
			item.Attributes.DefaultFormatter,
			item.Attributes.ScoreSortType,
			item.Attributes.SubmissionType,
			item.Attributes.Archived,
		)
	}
	return w.Flush()
}

func printGameCenterLeaderboardsMarkdown(resp *GameCenterLeaderboardsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Vendor ID | Formatter | Sort | Submission Type | Archived |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %s | %t |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(item.Attributes.VendorIdentifier),
			escapeMarkdown(item.Attributes.DefaultFormatter),
			escapeMarkdown(item.Attributes.ScoreSortType),
			escapeMarkdown(item.Attributes.SubmissionType),
			item.Attributes.Archived,
		)
	}
	return nil
}

func printGameCenterLeaderboardDeleteResultTable(result *GameCenterLeaderboardDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardDeleteResultMarkdown(result *GameCenterLeaderboardDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardSetsTable(resp *GameCenterLeaderboardSetsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tVendor ID")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.VendorIdentifier,
		)
	}
	return w.Flush()
}

func printGameCenterLeaderboardSetsMarkdown(resp *GameCenterLeaderboardSetsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Vendor ID |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(item.Attributes.VendorIdentifier),
		)
	}
	return nil
}

func printGameCenterLeaderboardSetDeleteResultTable(result *GameCenterLeaderboardSetDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardSetDeleteResultMarkdown(result *GameCenterLeaderboardSetDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardLocalizationsTable(resp *GameCenterLeaderboardLocalizationsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocale\tName\tFormatter Override\tFormatter Suffix\tFormatter Suffix Singular\tDescription")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Attributes.Locale,
			compactWhitespace(item.Attributes.Name),
			formatOptionalString(item.Attributes.FormatterOverride),
			formatOptionalString(item.Attributes.FormatterSuffix),
			formatOptionalString(item.Attributes.FormatterSuffixSingular),
			formatOptionalString(item.Attributes.Description),
		)
	}
	return w.Flush()
}

func printGameCenterLeaderboardLocalizationsMarkdown(resp *GameCenterLeaderboardLocalizationsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Locale | Name | Formatter Override | Formatter Suffix | Formatter Suffix Singular | Description |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Locale),
			escapeMarkdown(item.Attributes.Name),
			escapeMarkdown(formatOptionalString(item.Attributes.FormatterOverride)),
			escapeMarkdown(formatOptionalString(item.Attributes.FormatterSuffix)),
			escapeMarkdown(formatOptionalString(item.Attributes.FormatterSuffixSingular)),
			escapeMarkdown(formatOptionalString(item.Attributes.Description)),
		)
	}
	return nil
}

func printGameCenterLeaderboardLocalizationDeleteResultTable(result *GameCenterLeaderboardLocalizationDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardLocalizationDeleteResultMarkdown(result *GameCenterLeaderboardLocalizationDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardReleasesTable(resp *GameCenterLeaderboardReleasesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLive")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%t\n",
			item.ID,
			item.Attributes.Live,
		)
	}
	return w.Flush()
}

func printGameCenterLeaderboardReleasesMarkdown(resp *GameCenterLeaderboardReleasesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Live |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %t |\n",
			escapeMarkdown(item.ID),
			item.Attributes.Live,
		)
	}
	return nil
}

func printGameCenterLeaderboardReleaseDeleteResultTable(result *GameCenterLeaderboardReleaseDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardReleaseDeleteResultMarkdown(result *GameCenterLeaderboardReleaseDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterAchievementReleasesTable(resp *GameCenterAchievementReleasesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLive")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%t\n",
			item.ID,
			item.Attributes.Live,
		)
	}
	return w.Flush()
}

func printGameCenterAchievementReleasesMarkdown(resp *GameCenterAchievementReleasesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Live |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %t |\n",
			escapeMarkdown(item.ID),
			item.Attributes.Live,
		)
	}
	return nil
}

func printGameCenterAchievementReleaseDeleteResultTable(result *GameCenterAchievementReleaseDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterAchievementReleaseDeleteResultMarkdown(result *GameCenterAchievementReleaseDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardSetMembersUpdateResultTable(result *GameCenterLeaderboardSetMembersUpdateResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Set ID\tMember Count\tUpdated")
	fmt.Fprintf(w, "%s\t%d\t%t\n", result.SetID, result.MemberCount, result.Updated)
	return w.Flush()
}

func printGameCenterLeaderboardSetMembersUpdateResultMarkdown(result *GameCenterLeaderboardSetMembersUpdateResult) error {
	fmt.Fprintln(os.Stdout, "| Set ID | Member Count | Updated |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %d | %t |\n",
		escapeMarkdown(result.SetID),
		result.MemberCount,
		result.Updated,
	)
	return nil
}

func printGameCenterLeaderboardSetReleasesTable(resp *GameCenterLeaderboardSetReleasesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLive")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%t\n",
			item.ID,
			item.Attributes.Live,
		)
	}
	return w.Flush()
}

func printGameCenterLeaderboardSetReleasesMarkdown(resp *GameCenterLeaderboardSetReleasesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Live |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %t |\n",
			escapeMarkdown(item.ID),
			item.Attributes.Live,
		)
	}
	return nil
}

func printGameCenterLeaderboardSetReleaseDeleteResultTable(result *GameCenterLeaderboardSetReleaseDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardSetReleaseDeleteResultMarkdown(result *GameCenterLeaderboardSetReleaseDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardSetLocalizationsTable(resp *GameCenterLeaderboardSetLocalizationsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocale\tName")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			item.ID,
			item.Attributes.Locale,
			compactWhitespace(item.Attributes.Name),
		)
	}
	return w.Flush()
}

func printGameCenterLeaderboardSetLocalizationsMarkdown(resp *GameCenterLeaderboardSetLocalizationsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Locale | Name |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Locale),
			escapeMarkdown(item.Attributes.Name),
		)
	}
	return nil
}

func printGameCenterLeaderboardSetLocalizationDeleteResultTable(result *GameCenterLeaderboardSetLocalizationDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardSetLocalizationDeleteResultMarkdown(result *GameCenterLeaderboardSetLocalizationDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterAchievementLocalizationsTable(resp *GameCenterAchievementLocalizationsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocale\tName\tBefore Earned Description\tAfter Earned Description")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Attributes.Locale,
			compactWhitespace(item.Attributes.Name),
			compactWhitespace(item.Attributes.BeforeEarnedDescription),
			compactWhitespace(item.Attributes.AfterEarnedDescription),
		)
	}
	return w.Flush()
}

func printGameCenterAchievementLocalizationsMarkdown(resp *GameCenterAchievementLocalizationsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Locale | Name | Before Earned Description | After Earned Description |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Locale),
			escapeMarkdown(item.Attributes.Name),
			escapeMarkdown(item.Attributes.BeforeEarnedDescription),
			escapeMarkdown(item.Attributes.AfterEarnedDescription),
		)
	}
	return nil
}

func printGameCenterAchievementLocalizationDeleteResultTable(result *GameCenterAchievementLocalizationDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterAchievementLocalizationDeleteResultMarkdown(result *GameCenterAchievementLocalizationDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardImageUploadResultTable(result *GameCenterLeaderboardImageUploadResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocalization ID\tFile Name\tFile Size\tDelivery State\tUploaded")
	fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%t\n",
		result.ID,
		result.LocalizationID,
		result.FileName,
		result.FileSize,
		result.AssetDeliveryState,
		result.Uploaded,
	)
	return w.Flush()
}

func printGameCenterLeaderboardImageUploadResultMarkdown(result *GameCenterLeaderboardImageUploadResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Localization ID | File Name | File Size | Delivery State | Uploaded |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %d | %s | %t |\n",
		escapeMarkdown(result.ID),
		escapeMarkdown(result.LocalizationID),
		escapeMarkdown(result.FileName),
		result.FileSize,
		escapeMarkdown(result.AssetDeliveryState),
		result.Uploaded,
	)
	return nil
}

func printGameCenterLeaderboardImageDeleteResultTable(result *GameCenterLeaderboardImageDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardImageDeleteResultMarkdown(result *GameCenterLeaderboardImageDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterAchievementImageUploadResultTable(result *GameCenterAchievementImageUploadResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocalization ID\tFile Name\tFile Size\tDelivery State\tUploaded")
	fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%t\n",
		result.ID,
		result.LocalizationID,
		result.FileName,
		result.FileSize,
		result.AssetDeliveryState,
		result.Uploaded,
	)
	return w.Flush()
}

func printGameCenterAchievementImageUploadResultMarkdown(result *GameCenterAchievementImageUploadResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Localization ID | File Name | File Size | Delivery State | Uploaded |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %d | %s | %t |\n",
		escapeMarkdown(result.ID),
		escapeMarkdown(result.LocalizationID),
		escapeMarkdown(result.FileName),
		result.FileSize,
		escapeMarkdown(result.AssetDeliveryState),
		result.Uploaded,
	)
	return nil
}

func printGameCenterAchievementImageDeleteResultTable(result *GameCenterAchievementImageDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterAchievementImageDeleteResultMarkdown(result *GameCenterAchievementImageDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterLeaderboardSetImageUploadResultTable(result *GameCenterLeaderboardSetImageUploadResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocalization ID\tFile Name\tFile Size\tDelivery State\tUploaded")
	fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%t\n",
		result.ID,
		result.LocalizationID,
		result.FileName,
		result.FileSize,
		result.AssetDeliveryState,
		result.Uploaded,
	)
	return w.Flush()
}

func printGameCenterLeaderboardSetImageUploadResultMarkdown(result *GameCenterLeaderboardSetImageUploadResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Localization ID | File Name | File Size | Delivery State | Uploaded |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %d | %s | %t |\n",
		escapeMarkdown(result.ID),
		escapeMarkdown(result.LocalizationID),
		escapeMarkdown(result.FileName),
		result.FileSize,
		escapeMarkdown(result.AssetDeliveryState),
		result.Uploaded,
	)
	return nil
}

func printGameCenterLeaderboardSetImageDeleteResultTable(result *GameCenterLeaderboardSetImageDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterLeaderboardSetImageDeleteResultMarkdown(result *GameCenterLeaderboardSetImageDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}

func printGameCenterChallengesTable(resp *GameCenterChallengesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tVendor ID\tType\tRepeatable\tArchived")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%t\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.VendorIdentifier,
			item.Attributes.ChallengeType,
			item.Attributes.Repeatable,
			item.Attributes.Archived,
		)
	}
	return w.Flush()
}

func printGameCenterChallengesMarkdown(resp *GameCenterChallengesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Vendor ID | Type | Repeatable | Archived |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %t | %t |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(item.Attributes.VendorIdentifier),
			escapeMarkdown(item.Attributes.ChallengeType),
			item.Attributes.Repeatable,
			item.Attributes.Archived,
		)
	}
	return nil
}

func printGameCenterChallengeDeleteResultTable(result *GameCenterChallengeDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterChallengeDeleteResultMarkdown(result *GameCenterChallengeDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterChallengeVersionsTable(resp *GameCenterChallengeVersionsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVersion\tState")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%d\t%s\n",
			item.ID,
			item.Attributes.Version,
			item.Attributes.State,
		)
	}
	return w.Flush()
}

func printGameCenterChallengeVersionsMarkdown(resp *GameCenterChallengeVersionsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Version | State |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %d | %s |\n",
			escapeMarkdown(item.ID),
			item.Attributes.Version,
			escapeMarkdown(string(item.Attributes.State)),
		)
	}
	return nil
}

func printGameCenterChallengeLocalizationsTable(resp *GameCenterChallengeLocalizationsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocale\tName\tDescription")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			item.ID,
			item.Attributes.Locale,
			compactWhitespace(item.Attributes.Name),
			compactWhitespace(item.Attributes.Description),
		)
	}
	return w.Flush()
}

func printGameCenterChallengeLocalizationsMarkdown(resp *GameCenterChallengeLocalizationsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Locale | Name | Description |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Locale),
			escapeMarkdown(item.Attributes.Name),
			escapeMarkdown(item.Attributes.Description),
		)
	}
	return nil
}

func printGameCenterChallengeLocalizationDeleteResultTable(result *GameCenterChallengeLocalizationDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterChallengeLocalizationDeleteResultMarkdown(result *GameCenterChallengeLocalizationDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterChallengeImagesTable(resp *GameCenterChallengeImagesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFile Name\tFile Size\tDelivery State")
	for _, item := range resp.Data {
		state := ""
		if item.Attributes.AssetDeliveryState != nil {
			state = item.Attributes.AssetDeliveryState.State
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			item.ID,
			item.Attributes.FileName,
			item.Attributes.FileSize,
			state,
		)
	}
	return w.Flush()
}

func printGameCenterChallengeImagesMarkdown(resp *GameCenterChallengeImagesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | File Name | File Size | Delivery State |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		state := ""
		if item.Attributes.AssetDeliveryState != nil {
			state = item.Attributes.AssetDeliveryState.State
		}
		fmt.Fprintf(os.Stdout, "| %s | %s | %d | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.FileName),
			item.Attributes.FileSize,
			escapeMarkdown(state),
		)
	}
	return nil
}

func printGameCenterChallengeImageUploadResultTable(result *GameCenterChallengeImageUploadResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocalization ID\tFile Name\tFile Size\tDelivery State\tUploaded")
	fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%t\n",
		result.ID,
		result.LocalizationID,
		result.FileName,
		result.FileSize,
		result.AssetDeliveryState,
		result.Uploaded,
	)
	return w.Flush()
}

func printGameCenterChallengeImageUploadResultMarkdown(result *GameCenterChallengeImageUploadResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Localization ID | File Name | File Size | Delivery State | Uploaded |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %d | %s | %t |\n",
		escapeMarkdown(result.ID),
		escapeMarkdown(result.LocalizationID),
		escapeMarkdown(result.FileName),
		result.FileSize,
		escapeMarkdown(result.AssetDeliveryState),
		result.Uploaded,
	)
	return nil
}

func printGameCenterChallengeImageDeleteResultTable(result *GameCenterChallengeImageDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterChallengeImageDeleteResultMarkdown(result *GameCenterChallengeImageDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterChallengeReleasesTable(resp *GameCenterChallengeVersionReleasesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\n", item.ID)
	}
	return w.Flush()
}

func printGameCenterChallengeReleasesMarkdown(resp *GameCenterChallengeVersionReleasesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID |")
	fmt.Fprintln(os.Stdout, "| --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s |\n", escapeMarkdown(item.ID))
	}
	return nil
}

func printGameCenterChallengeReleaseDeleteResultTable(result *GameCenterChallengeVersionReleaseDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterChallengeReleaseDeleteResultMarkdown(result *GameCenterChallengeVersionReleaseDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterActivitiesTable(resp *GameCenterActivitiesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tVendor ID\tPlay Style\tMin Players\tMax Players\tParty Code\tArchived")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%t\t%t\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.VendorIdentifier,
			item.Attributes.PlayStyle,
			item.Attributes.MinimumPlayersCount,
			item.Attributes.MaximumPlayersCount,
			item.Attributes.SupportsPartyCode,
			item.Attributes.Archived,
		)
	}
	return w.Flush()
}

func printGameCenterActivitiesMarkdown(resp *GameCenterActivitiesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Vendor ID | Play Style | Min Players | Max Players | Party Code | Archived |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %d | %d | %t | %t |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(item.Attributes.VendorIdentifier),
			escapeMarkdown(item.Attributes.PlayStyle),
			item.Attributes.MinimumPlayersCount,
			item.Attributes.MaximumPlayersCount,
			item.Attributes.SupportsPartyCode,
			item.Attributes.Archived,
		)
	}
	return nil
}

func printGameCenterActivityDeleteResultTable(result *GameCenterActivityDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterActivityDeleteResultMarkdown(result *GameCenterActivityDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterActivityVersionsTable(resp *GameCenterActivityVersionsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVersion\tState\tFallback URL")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			item.ID,
			item.Attributes.Version,
			item.Attributes.State,
			item.Attributes.FallbackURL,
		)
	}
	return w.Flush()
}

func printGameCenterActivityVersionsMarkdown(resp *GameCenterActivityVersionsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Version | State | Fallback URL |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %d | %s | %s |\n",
			escapeMarkdown(item.ID),
			item.Attributes.Version,
			escapeMarkdown(string(item.Attributes.State)),
			escapeMarkdown(item.Attributes.FallbackURL),
		)
	}
	return nil
}

func printGameCenterActivityLocalizationsTable(resp *GameCenterActivityLocalizationsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocale\tName\tDescription")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			item.ID,
			item.Attributes.Locale,
			compactWhitespace(item.Attributes.Name),
			compactWhitespace(item.Attributes.Description),
		)
	}
	return w.Flush()
}

func printGameCenterActivityLocalizationsMarkdown(resp *GameCenterActivityLocalizationsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Locale | Name | Description |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Locale),
			escapeMarkdown(item.Attributes.Name),
			escapeMarkdown(item.Attributes.Description),
		)
	}
	return nil
}

func printGameCenterActivityLocalizationDeleteResultTable(result *GameCenterActivityLocalizationDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterActivityLocalizationDeleteResultMarkdown(result *GameCenterActivityLocalizationDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterActivityImagesTable(resp *GameCenterActivityImagesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFile Name\tFile Size\tDelivery State")
	for _, item := range resp.Data {
		state := ""
		if item.Attributes.AssetDeliveryState != nil {
			state = item.Attributes.AssetDeliveryState.State
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			item.ID,
			item.Attributes.FileName,
			item.Attributes.FileSize,
			state,
		)
	}
	return w.Flush()
}

func printGameCenterActivityImagesMarkdown(resp *GameCenterActivityImagesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | File Name | File Size | Delivery State |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		state := ""
		if item.Attributes.AssetDeliveryState != nil {
			state = item.Attributes.AssetDeliveryState.State
		}
		fmt.Fprintf(os.Stdout, "| %s | %s | %d | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.FileName),
			item.Attributes.FileSize,
			escapeMarkdown(state),
		)
	}
	return nil
}

func printGameCenterActivityImageUploadResultTable(result *GameCenterActivityImageUploadResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLocalization ID\tFile Name\tFile Size\tDelivery State\tUploaded")
	fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%t\n",
		result.ID,
		result.LocalizationID,
		result.FileName,
		result.FileSize,
		result.AssetDeliveryState,
		result.Uploaded,
	)
	return w.Flush()
}

func printGameCenterActivityImageUploadResultMarkdown(result *GameCenterActivityImageUploadResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Localization ID | File Name | File Size | Delivery State | Uploaded |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %d | %s | %t |\n",
		escapeMarkdown(result.ID),
		escapeMarkdown(result.LocalizationID),
		escapeMarkdown(result.FileName),
		result.FileSize,
		escapeMarkdown(result.AssetDeliveryState),
		result.Uploaded,
	)
	return nil
}

func printGameCenterActivityImageDeleteResultTable(result *GameCenterActivityImageDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterActivityImageDeleteResultMarkdown(result *GameCenterActivityImageDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterActivityReleasesTable(resp *GameCenterActivityVersionReleasesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\n", item.ID)
	}
	return w.Flush()
}

func printGameCenterActivityReleasesMarkdown(resp *GameCenterActivityVersionReleasesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID |")
	fmt.Fprintln(os.Stdout, "| --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s |\n", escapeMarkdown(item.ID))
	}
	return nil
}

func printGameCenterActivityReleaseDeleteResultTable(result *GameCenterActivityVersionReleaseDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterActivityReleaseDeleteResultMarkdown(result *GameCenterActivityVersionReleaseDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterGroupsTable(resp *GameCenterGroupsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
		)
	}
	return w.Flush()
}

func printGameCenterGroupsMarkdown(resp *GameCenterGroupsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
		)
	}
	return nil
}

func printGameCenterGroupDeleteResultTable(result *GameCenterGroupDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterGroupDeleteResultMarkdown(result *GameCenterGroupDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterMatchmakingQueuesTable(resp *GameCenterMatchmakingQueuesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tClassic Bundle IDs")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			formatStringList(item.Attributes.ClassicMatchmakingBundleIDs),
		)
	}
	return w.Flush()
}

func printGameCenterMatchmakingQueuesMarkdown(resp *GameCenterMatchmakingQueuesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Classic Bundle IDs |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(formatStringList(item.Attributes.ClassicMatchmakingBundleIDs)),
		)
	}
	return nil
}

func printGameCenterMatchmakingQueueDeleteResultTable(result *GameCenterMatchmakingQueueDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterMatchmakingQueueDeleteResultMarkdown(result *GameCenterMatchmakingQueueDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterMatchmakingRuleSetsTable(resp *GameCenterMatchmakingRuleSetsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tLanguage\tMin Players\tMax Players")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.RuleLanguageVersion,
			item.Attributes.MinPlayers,
			item.Attributes.MaxPlayers,
		)
	}
	return w.Flush()
}

func printGameCenterMatchmakingRuleSetsMarkdown(resp *GameCenterMatchmakingRuleSetsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Language | Min Players | Max Players |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %d | %d | %d |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			item.Attributes.RuleLanguageVersion,
			item.Attributes.MinPlayers,
			item.Attributes.MaxPlayers,
		)
	}
	return nil
}

func printGameCenterMatchmakingRuleSetDeleteResultTable(result *GameCenterMatchmakingRuleSetDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterMatchmakingRuleSetDeleteResultMarkdown(result *GameCenterMatchmakingRuleSetDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterMatchmakingRulesTable(resp *GameCenterMatchmakingRulesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tType\tWeight")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%g\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.Type,
			item.Attributes.Weight,
		)
	}
	return w.Flush()
}

func printGameCenterMatchmakingRulesMarkdown(resp *GameCenterMatchmakingRulesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Type | Weight |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %g |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(item.Attributes.Type),
			item.Attributes.Weight,
		)
	}
	return nil
}

func printGameCenterMatchmakingRuleDeleteResultTable(result *GameCenterMatchmakingRuleDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterMatchmakingRuleDeleteResultMarkdown(result *GameCenterMatchmakingRuleDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterMatchmakingTeamsTable(resp *GameCenterMatchmakingTeamsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tReference Name\tMin Players\tMax Players")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\n",
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.MinPlayers,
			item.Attributes.MaxPlayers,
		)
	}
	return w.Flush()
}

func printGameCenterMatchmakingTeamsMarkdown(resp *GameCenterMatchmakingTeamsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Reference Name | Min Players | Max Players |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %d | %d |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.ReferenceName),
			item.Attributes.MinPlayers,
			item.Attributes.MaxPlayers,
		)
	}
	return nil
}

func printGameCenterMatchmakingTeamDeleteResultTable(result *GameCenterMatchmakingTeamDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printGameCenterMatchmakingTeamDeleteResultMarkdown(result *GameCenterMatchmakingTeamDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printGameCenterMetricsTable(resp *GameCenterMetricsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Start\tEnd\tGranularity\tValues\tDimensions")
	for _, item := range resp.Data {
		for _, point := range item.DataPoints {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				point.Start,
				point.End,
				formatMetricGranularity(item.Granularity),
				formatMetricJSON(point.Values),
				formatMetricJSON(item.Dimensions),
			)
		}
	}
	return w.Flush()
}

func printGameCenterMetricsMarkdown(resp *GameCenterMetricsResponse) error {
	fmt.Fprintln(os.Stdout, "| Start | End | Granularity | Values | Dimensions |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		for _, point := range item.DataPoints {
			fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s |\n",
				escapeMarkdown(point.Start),
				escapeMarkdown(point.End),
				escapeMarkdown(formatMetricGranularity(item.Granularity)),
				escapeMarkdown(formatMetricJSON(point.Values)),
				escapeMarkdown(formatMetricJSON(item.Dimensions)),
			)
		}
	}
	return nil
}

func printGameCenterMatchmakingRuleSetTestTable(resp *GameCenterMatchmakingRuleSetTestResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID")
	fmt.Fprintf(w, "%s\n", resp.Data.ID)
	return w.Flush()
}

func printGameCenterMatchmakingRuleSetTestMarkdown(resp *GameCenterMatchmakingRuleSetTestResponse) error {
	fmt.Fprintln(os.Stdout, "| ID |")
	fmt.Fprintln(os.Stdout, "| --- |")
	fmt.Fprintf(os.Stdout, "| %s |\n", escapeMarkdown(resp.Data.ID))
	return nil
}

func printGameCenterLeaderboardEntrySubmissionTable(resp *GameCenterLeaderboardEntrySubmissionResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVendor ID\tScore\tBundle ID\tScoped Player ID\tSubmitted Date")
	attrs := resp.Data.Attributes
	submittedDate := ""
	if attrs.SubmittedDate != nil {
		submittedDate = *attrs.SubmittedDate
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		resp.Data.ID,
		compactWhitespace(attrs.VendorIdentifier),
		compactWhitespace(attrs.Score),
		compactWhitespace(attrs.BundleID),
		compactWhitespace(attrs.ScopedPlayerID),
		compactWhitespace(submittedDate),
	)
	return w.Flush()
}

func printGameCenterLeaderboardEntrySubmissionMarkdown(resp *GameCenterLeaderboardEntrySubmissionResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Vendor ID | Score | Bundle ID | Scoped Player ID | Submitted Date |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	attrs := resp.Data.Attributes
	submittedDate := ""
	if attrs.SubmittedDate != nil {
		submittedDate = *attrs.SubmittedDate
	}
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %s |\n",
		escapeMarkdown(resp.Data.ID),
		escapeMarkdown(attrs.VendorIdentifier),
		escapeMarkdown(attrs.Score),
		escapeMarkdown(attrs.BundleID),
		escapeMarkdown(attrs.ScopedPlayerID),
		escapeMarkdown(submittedDate),
	)
	return nil
}

func printGameCenterPlayerAchievementSubmissionTable(resp *GameCenterPlayerAchievementSubmissionResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVendor ID\tPercent\tBundle ID\tScoped Player ID\tSubmitted Date")
	attrs := resp.Data.Attributes
	submittedDate := ""
	if attrs.SubmittedDate != nil {
		submittedDate = *attrs.SubmittedDate
	}
	fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
		resp.Data.ID,
		compactWhitespace(attrs.VendorIdentifier),
		attrs.PercentageAchieved,
		compactWhitespace(attrs.BundleID),
		compactWhitespace(attrs.ScopedPlayerID),
		compactWhitespace(submittedDate),
	)
	return w.Flush()
}

func printGameCenterPlayerAchievementSubmissionMarkdown(resp *GameCenterPlayerAchievementSubmissionResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Vendor ID | Percent | Bundle ID | Scoped Player ID | Submitted Date |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	attrs := resp.Data.Attributes
	submittedDate := ""
	if attrs.SubmittedDate != nil {
		submittedDate = *attrs.SubmittedDate
	}
	fmt.Fprintf(os.Stdout, "| %s | %s | %d | %s | %s | %s |\n",
		escapeMarkdown(resp.Data.ID),
		escapeMarkdown(attrs.VendorIdentifier),
		attrs.PercentageAchieved,
		escapeMarkdown(attrs.BundleID),
		escapeMarkdown(attrs.ScopedPlayerID),
		escapeMarkdown(submittedDate),
	)
	return nil
}

func formatStringList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ",")
}

func formatMetricJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func formatMetricGranularity(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
