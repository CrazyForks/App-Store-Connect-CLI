package asc

import (
	"encoding/json"
	"os"
)

// PrintJSON prints data as minified JSON (best for AI agents)
func PrintJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(data)
}

// PrintPrettyJSON prints data as indented JSON (best for debugging).
func PrintPrettyJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintMarkdown prints data as Markdown table
func PrintMarkdown(data interface{}) error {
	switch v := data.(type) {
	case *FeedbackResponse:
		return printFeedbackMarkdown(v)
	case *CrashesResponse:
		return printCrashesMarkdown(v)
	case *ReviewsResponse:
		return printReviewsMarkdown(v)
	case *AppsResponse:
		return printAppsMarkdown(v)
	case *AppCategoriesResponse:
		return printAppCategoriesMarkdown(v)
	case *AppResponse:
		return printAppsMarkdown(&AppsResponse{Data: []Resource[AppAttributes]{v.Data}})
	case *InAppPurchasesV2Response:
		return printInAppPurchasesMarkdown(v)
	case *InAppPurchaseV2Response:
		return printInAppPurchasesMarkdown(&InAppPurchasesV2Response{Data: []Resource[InAppPurchaseV2Attributes]{v.Data}})
	case *InAppPurchaseLocalizationsResponse:
		return printInAppPurchaseLocalizationsMarkdown(v)
	case *SubscriptionGroupsResponse:
		return printSubscriptionGroupsMarkdown(v)
	case *SubscriptionGroupResponse:
		return printSubscriptionGroupsMarkdown(&SubscriptionGroupsResponse{Data: []Resource[SubscriptionGroupAttributes]{v.Data}})
	case *SubscriptionsResponse:
		return printSubscriptionsMarkdown(v)
	case *SubscriptionResponse:
		return printSubscriptionsMarkdown(&SubscriptionsResponse{Data: []Resource[SubscriptionAttributes]{v.Data}})
	case *SubscriptionPriceResponse:
		return printSubscriptionPriceMarkdown(v)
	case *SubscriptionAvailabilityResponse:
		return printSubscriptionAvailabilityMarkdown(v)
	case *TerritoriesResponse:
		return printTerritoriesMarkdown(v)
	case *AppPricePointsV3Response:
		return printAppPricePointsMarkdown(v)
	case *AppPriceScheduleResponse:
		return printAppPriceScheduleMarkdown(v)
	case *AppPricesResponse:
		return printAppPricesMarkdown(v)
	case *BuildsResponse:
		return printBuildsMarkdown(v)
	case *AppStoreVersionsResponse:
		return printAppStoreVersionsMarkdown(v)
	case *PreReleaseVersionsResponse:
		return printPreReleaseVersionsMarkdown(v)
	case *BuildResponse:
		return printBuildsMarkdown(&BuildsResponse{Data: []Resource[BuildAttributes]{v.Data}})
	case *AppAvailabilityV2Response:
		return printAppAvailabilityMarkdown(v)
	case *TerritoryAvailabilitiesResponse:
		return printTerritoryAvailabilitiesMarkdown(v)
	case *PreReleaseVersionResponse:
		return printPreReleaseVersionsMarkdown(&PreReleaseVersionsResponse{Data: []PreReleaseVersion{v.Data}})
	case *AppStoreVersionLocalizationsResponse:
		return printAppStoreVersionLocalizationsMarkdown(v)
	case *AppStoreVersionLocalizationResponse:
		return printAppStoreVersionLocalizationsMarkdown(&AppStoreVersionLocalizationsResponse{Data: []Resource[AppStoreVersionLocalizationAttributes]{v.Data}})
	case *AppInfoLocalizationsResponse:
		return printAppInfoLocalizationsMarkdown(v)
	case *BetaGroupsResponse:
		return printBetaGroupsMarkdown(v)
	case *BetaGroupResponse:
		return printBetaGroupsMarkdown(&BetaGroupsResponse{Data: []Resource[BetaGroupAttributes]{v.Data}})
	case *BetaTestersResponse:
		return printBetaTestersMarkdown(v)
	case *BetaTesterResponse:
		return printBetaTesterMarkdown(v)
	case *UsersResponse:
		return printUsersMarkdown(v)
	case *UserResponse:
		return printUsersMarkdown(&UsersResponse{Data: []Resource[UserAttributes]{v.Data}})
	case *UserInvitationsResponse:
		return printUserInvitationsMarkdown(v)
	case *UserInvitationResponse:
		return printUserInvitationsMarkdown(&UserInvitationsResponse{Data: []Resource[UserInvitationAttributes]{v.Data}})
	case *UserDeleteResult:
		return printUserDeleteResultMarkdown(v)
	case *UserInvitationRevokeResult:
		return printUserInvitationRevokeResultMarkdown(v)
	case *BetaAppReviewDetailsResponse:
		return printBetaAppReviewDetailsMarkdown(v)
	case *BetaAppReviewDetailResponse:
		return printBetaAppReviewDetailMarkdown(v)
	case *BetaAppReviewSubmissionsResponse:
		return printBetaAppReviewSubmissionsMarkdown(v)
	case *BetaAppReviewSubmissionResponse:
		return printBetaAppReviewSubmissionMarkdown(v)
	case *BuildBetaDetailsResponse:
		return printBuildBetaDetailsMarkdown(v)
	case *BuildBetaDetailResponse:
		return printBuildBetaDetailMarkdown(v)
	case *BetaRecruitmentCriterionOptionsResponse:
		return printBetaRecruitmentCriterionOptionsMarkdown(v)
	case *BetaRecruitmentCriteriaResponse:
		return printBetaRecruitmentCriteriaMarkdown(v)
	case *Response[BetaGroupMetricAttributes]:
		return printBetaGroupMetricsMarkdown(v.Data)
	case *SandboxTestersResponse:
		return printSandboxTestersMarkdown(v)
	case *SandboxTesterResponse:
		return printSandboxTestersMarkdown(&SandboxTestersResponse{Data: []Resource[SandboxTesterAttributes]{v.Data}})
	case *LocalizationDownloadResult:
		return printLocalizationDownloadResultMarkdown(v)
	case *LocalizationUploadResult:
		return printLocalizationUploadResultMarkdown(v)
	case *BuildUploadResult:
		return printBuildUploadResultMarkdown(v)
	case *TestFlightPublishResult:
		return printTestFlightPublishResultMarkdown(v)
	case *AppStorePublishResult:
		return printAppStorePublishResultMarkdown(v)
	case *SalesReportResult:
		return printSalesReportResultMarkdown(v)
	case *FinanceReportResult:
		return printFinanceReportResultMarkdown(v)
	case *FinanceRegionsResult:
		return printFinanceRegionsMarkdown(v)
	case *AnalyticsReportRequestResult:
		return printAnalyticsReportRequestResultMarkdown(v)
	case *AnalyticsReportRequestsResponse:
		return printAnalyticsReportRequestsMarkdown(v)
	case *AnalyticsReportRequestResponse:
		return printAnalyticsReportRequestsMarkdown(&AnalyticsReportRequestsResponse{Data: []AnalyticsReportRequestResource{v.Data}, Links: v.Links})
	case *AnalyticsReportDownloadResult:
		return printAnalyticsReportDownloadResultMarkdown(v)
	case *AnalyticsReportGetResult:
		return printAnalyticsReportGetResultMarkdown(v)
	case *AppStoreVersionSubmissionResult:
		return printAppStoreVersionSubmissionMarkdown(v)
	case *AppStoreVersionSubmissionCreateResult:
		return printAppStoreVersionSubmissionCreateMarkdown(v)
	case *AppStoreVersionSubmissionStatusResult:
		return printAppStoreVersionSubmissionStatusMarkdown(v)
	case *AppStoreVersionSubmissionCancelResult:
		return printAppStoreVersionSubmissionCancelMarkdown(v)
	case *AppStoreVersionDetailResult:
		return printAppStoreVersionDetailMarkdown(v)
	case *AppStoreVersionAttachBuildResult:
		return printAppStoreVersionAttachBuildMarkdown(v)
	case *AppStoreVersionPhasedReleaseResponse:
		return printAppStoreVersionPhasedReleaseMarkdown(v)
	case *AppStoreVersionPhasedReleaseDeleteResult:
		return printAppStoreVersionPhasedReleaseDeleteResultMarkdown(v)
	case *BuildBetaGroupsUpdateResult:
		return printBuildBetaGroupsUpdateMarkdown(v)
	case *InAppPurchaseDeleteResult:
		return printInAppPurchaseDeleteResultMarkdown(v)
	case *SubscriptionGroupDeleteResult:
		return printSubscriptionGroupDeleteResultMarkdown(v)
	case *SubscriptionDeleteResult:
		return printSubscriptionDeleteResultMarkdown(v)
	case *BetaTesterDeleteResult:
		return printBetaTesterDeleteResultMarkdown(v)
	case *BetaTesterGroupsUpdateResult:
		return printBetaTesterGroupsUpdateResultMarkdown(v)
	case *AppStoreVersionLocalizationDeleteResult:
		return printAppStoreVersionLocalizationDeleteResultMarkdown(v)
	case *BetaTesterInvitationResult:
		return printBetaTesterInvitationResultMarkdown(v)
	case *SandboxTesterDeleteResult:
		return printSandboxTesterDeleteResultMarkdown(v)
	case *SandboxTesterClearHistoryResult:
		return printSandboxTesterClearHistoryResultMarkdown(v)
	case *XcodeCloudRunResult:
		return printXcodeCloudRunResultMarkdown(v)
	case *XcodeCloudStatusResult:
		return printXcodeCloudStatusResultMarkdown(v)
	case *CiProductsResponse:
		return printCiProductsMarkdown(v)
	case *CiWorkflowsResponse:
		return printCiWorkflowsMarkdown(v)
	case *CiBuildRunsResponse:
		return printCiBuildRunsMarkdown(v)
	case *CiBuildActionsResponse:
		return printCiBuildActionsMarkdown(v)
	case *CustomerReviewResponseResponse:
		return printCustomerReviewResponseMarkdown(v)
	case *CustomerReviewResponseDeleteResult:
		return printCustomerReviewResponseDeleteResultMarkdown(v)
	default:
		return PrintJSON(data)
	}
}

// PrintTable prints data as a formatted table
func PrintTable(data interface{}) error {
	switch v := data.(type) {
	case *FeedbackResponse:
		return printFeedbackTable(v)
	case *CrashesResponse:
		return printCrashesTable(v)
	case *ReviewsResponse:
		return printReviewsTable(v)
	case *AppsResponse:
		return printAppsTable(v)
	case *AppCategoriesResponse:
		return printAppCategoriesTable(v)
	case *AppResponse:
		return printAppsTable(&AppsResponse{Data: []Resource[AppAttributes]{v.Data}})
	case *InAppPurchasesV2Response:
		return printInAppPurchasesTable(v)
	case *InAppPurchaseV2Response:
		return printInAppPurchasesTable(&InAppPurchasesV2Response{Data: []Resource[InAppPurchaseV2Attributes]{v.Data}})
	case *InAppPurchaseLocalizationsResponse:
		return printInAppPurchaseLocalizationsTable(v)
	case *SubscriptionGroupsResponse:
		return printSubscriptionGroupsTable(v)
	case *SubscriptionGroupResponse:
		return printSubscriptionGroupsTable(&SubscriptionGroupsResponse{Data: []Resource[SubscriptionGroupAttributes]{v.Data}})
	case *SubscriptionsResponse:
		return printSubscriptionsTable(v)
	case *SubscriptionResponse:
		return printSubscriptionsTable(&SubscriptionsResponse{Data: []Resource[SubscriptionAttributes]{v.Data}})
	case *SubscriptionPriceResponse:
		return printSubscriptionPriceTable(v)
	case *SubscriptionAvailabilityResponse:
		return printSubscriptionAvailabilityTable(v)
	case *TerritoriesResponse:
		return printTerritoriesTable(v)
	case *AppPricePointsV3Response:
		return printAppPricePointsTable(v)
	case *AppPriceScheduleResponse:
		return printAppPriceScheduleTable(v)
	case *AppPricesResponse:
		return printAppPricesTable(v)
	case *BuildsResponse:
		return printBuildsTable(v)
	case *AppStoreVersionsResponse:
		return printAppStoreVersionsTable(v)
	case *PreReleaseVersionsResponse:
		return printPreReleaseVersionsTable(v)
	case *BuildResponse:
		return printBuildsTable(&BuildsResponse{Data: []Resource[BuildAttributes]{v.Data}})
	case *AppAvailabilityV2Response:
		return printAppAvailabilityTable(v)
	case *TerritoryAvailabilitiesResponse:
		return printTerritoryAvailabilitiesTable(v)
	case *PreReleaseVersionResponse:
		return printPreReleaseVersionsTable(&PreReleaseVersionsResponse{Data: []PreReleaseVersion{v.Data}})
	case *AppStoreVersionLocalizationsResponse:
		return printAppStoreVersionLocalizationsTable(v)
	case *AppStoreVersionLocalizationResponse:
		return printAppStoreVersionLocalizationsTable(&AppStoreVersionLocalizationsResponse{Data: []Resource[AppStoreVersionLocalizationAttributes]{v.Data}})
	case *AppInfoLocalizationsResponse:
		return printAppInfoLocalizationsTable(v)
	case *BetaGroupsResponse:
		return printBetaGroupsTable(v)
	case *BetaGroupResponse:
		return printBetaGroupsTable(&BetaGroupsResponse{Data: []Resource[BetaGroupAttributes]{v.Data}})
	case *BetaTestersResponse:
		return printBetaTestersTable(v)
	case *BetaTesterResponse:
		return printBetaTesterTable(v)
	case *UsersResponse:
		return printUsersTable(v)
	case *UserResponse:
		return printUsersTable(&UsersResponse{Data: []Resource[UserAttributes]{v.Data}})
	case *UserInvitationsResponse:
		return printUserInvitationsTable(v)
	case *UserInvitationResponse:
		return printUserInvitationsTable(&UserInvitationsResponse{Data: []Resource[UserInvitationAttributes]{v.Data}})
	case *UserDeleteResult:
		return printUserDeleteResultTable(v)
	case *UserInvitationRevokeResult:
		return printUserInvitationRevokeResultTable(v)
	case *BetaAppReviewDetailsResponse:
		return printBetaAppReviewDetailsTable(v)
	case *BetaAppReviewDetailResponse:
		return printBetaAppReviewDetailTable(v)
	case *BetaAppReviewSubmissionsResponse:
		return printBetaAppReviewSubmissionsTable(v)
	case *BetaAppReviewSubmissionResponse:
		return printBetaAppReviewSubmissionTable(v)
	case *BuildBetaDetailsResponse:
		return printBuildBetaDetailsTable(v)
	case *BuildBetaDetailResponse:
		return printBuildBetaDetailTable(v)
	case *BetaRecruitmentCriterionOptionsResponse:
		return printBetaRecruitmentCriterionOptionsTable(v)
	case *BetaRecruitmentCriteriaResponse:
		return printBetaRecruitmentCriteriaTable(v)
	case *Response[BetaGroupMetricAttributes]:
		return printBetaGroupMetricsTable(v.Data)
	case *SandboxTestersResponse:
		return printSandboxTestersTable(v)
	case *SandboxTesterResponse:
		return printSandboxTestersTable(&SandboxTestersResponse{Data: []Resource[SandboxTesterAttributes]{v.Data}})
	case *LocalizationDownloadResult:
		return printLocalizationDownloadResultTable(v)
	case *LocalizationUploadResult:
		return printLocalizationUploadResultTable(v)
	case *BuildUploadResult:
		return printBuildUploadResultTable(v)
	case *TestFlightPublishResult:
		return printTestFlightPublishResultTable(v)
	case *AppStorePublishResult:
		return printAppStorePublishResultTable(v)
	case *SalesReportResult:
		return printSalesReportResultTable(v)
	case *FinanceReportResult:
		return printFinanceReportResultTable(v)
	case *FinanceRegionsResult:
		return printFinanceRegionsTable(v)
	case *AnalyticsReportRequestResult:
		return printAnalyticsReportRequestResultTable(v)
	case *AnalyticsReportRequestsResponse:
		return printAnalyticsReportRequestsTable(v)
	case *AnalyticsReportRequestResponse:
		return printAnalyticsReportRequestsTable(&AnalyticsReportRequestsResponse{Data: []AnalyticsReportRequestResource{v.Data}, Links: v.Links})
	case *AnalyticsReportDownloadResult:
		return printAnalyticsReportDownloadResultTable(v)
	case *AnalyticsReportGetResult:
		return printAnalyticsReportGetResultTable(v)
	case *AppStoreVersionSubmissionResult:
		return printAppStoreVersionSubmissionTable(v)
	case *AppStoreVersionSubmissionCreateResult:
		return printAppStoreVersionSubmissionCreateTable(v)
	case *AppStoreVersionSubmissionStatusResult:
		return printAppStoreVersionSubmissionStatusTable(v)
	case *AppStoreVersionSubmissionCancelResult:
		return printAppStoreVersionSubmissionCancelTable(v)
	case *AppStoreVersionDetailResult:
		return printAppStoreVersionDetailTable(v)
	case *AppStoreVersionAttachBuildResult:
		return printAppStoreVersionAttachBuildTable(v)
	case *AppStoreVersionPhasedReleaseResponse:
		return printAppStoreVersionPhasedReleaseTable(v)
	case *AppStoreVersionPhasedReleaseDeleteResult:
		return printAppStoreVersionPhasedReleaseDeleteResultTable(v)
	case *BuildBetaGroupsUpdateResult:
		return printBuildBetaGroupsUpdateTable(v)
	case *InAppPurchaseDeleteResult:
		return printInAppPurchaseDeleteResultTable(v)
	case *SubscriptionGroupDeleteResult:
		return printSubscriptionGroupDeleteResultTable(v)
	case *SubscriptionDeleteResult:
		return printSubscriptionDeleteResultTable(v)
	case *BetaTesterDeleteResult:
		return printBetaTesterDeleteResultTable(v)
	case *BetaTesterGroupsUpdateResult:
		return printBetaTesterGroupsUpdateResultTable(v)
	case *AppStoreVersionLocalizationDeleteResult:
		return printAppStoreVersionLocalizationDeleteResultTable(v)
	case *BetaTesterInvitationResult:
		return printBetaTesterInvitationResultTable(v)
	case *SandboxTesterDeleteResult:
		return printSandboxTesterDeleteResultTable(v)
	case *SandboxTesterClearHistoryResult:
		return printSandboxTesterClearHistoryResultTable(v)
	case *XcodeCloudRunResult:
		return printXcodeCloudRunResultTable(v)
	case *XcodeCloudStatusResult:
		return printXcodeCloudStatusResultTable(v)
	case *CiProductsResponse:
		return printCiProductsTable(v)
	case *CiWorkflowsResponse:
		return printCiWorkflowsTable(v)
	case *CiBuildRunsResponse:
		return printCiBuildRunsTable(v)
	case *CiBuildActionsResponse:
		return printCiBuildActionsTable(v)
	case *CustomerReviewResponseResponse:
		return printCustomerReviewResponseTable(v)
	case *CustomerReviewResponseDeleteResult:
		return printCustomerReviewResponseDeleteResultTable(v)
	default:
		return PrintJSON(data)
	}
}
