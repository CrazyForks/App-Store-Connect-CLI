package asc

import (
	"fmt"
	"os"
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
