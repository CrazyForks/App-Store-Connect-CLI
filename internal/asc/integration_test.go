//go:build integration

package asc

import (
	"context"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestIntegrationEndpoints(t *testing.T) {
	keyID := os.Getenv("ASC_KEY_ID")
	issuerID := os.Getenv("ASC_ISSUER_ID")
	keyPath := os.Getenv("ASC_PRIVATE_KEY_PATH")
	appID := os.Getenv("ASC_APP_ID")

	if keyID == "" || issuerID == "" || keyPath == "" || appID == "" {
		t.Skip("integration tests require ASC_KEY_ID, ASC_ISSUER_ID, ASC_PRIVATE_KEY_PATH, ASC_APP_ID")
	}

	client, err := NewClient(keyID, issuerID, keyPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	t.Run("app_store_version_localizations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		versions, err := client.GetAppStoreVersions(ctx, appID, WithAppStoreVersionsLimit(50))
		if err != nil {
			t.Fatalf("failed to fetch app store versions: %v", err)
		}
		if versions == nil {
			t.Fatal("expected app store versions response")
		}
		if len(versions.Data) == 0 {
			t.Skip("no app store versions available")
		}

		selected := selectLatestAppStoreVersionForTest(versions.Data)
		if strings.TrimSpace(selected.ID) == "" {
			t.Skip("no app store version id available")
		}

		localizations, err := client.GetAppStoreVersionLocalizations(ctx, selected.ID, WithAppStoreVersionLocalizationsLimit(1))
		if err != nil {
			t.Fatalf("failed to fetch app store version localizations: %v", err)
		}
		if localizations == nil {
			t.Fatal("expected localizations response")
		}
		if len(localizations.Data) == 0 {
			t.Skip("no app store version localizations available")
		}
		assertASCLink(t, localizations.Links.Self)
		assertASCLink(t, localizations.Links.Next)
	})

	t.Run("feedback", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		feedback, err := client.GetFeedback(ctx, appID, WithFeedbackLimit(1))
		if err != nil {
			t.Fatalf("failed to fetch feedback: %v", err)
		}
		if feedback == nil {
			t.Fatal("expected feedback response")
		}
		assertLimit(t, len(feedback.Data), 1)
		assertASCLink(t, feedback.Links.Self)
		assertASCLink(t, feedback.Links.Next)
		if len(feedback.Data) > 0 && feedback.Data[0].Type == "" {
			t.Fatal("expected feedback data type to be set")
		}
		if feedback.Links.Next != "" {
			nextCtx, nextCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer nextCancel()
			nextFeedback, err := client.GetFeedback(nextCtx, appID, WithFeedbackNextURL(feedback.Links.Next))
			if err != nil {
				t.Fatalf("failed to fetch feedback next page: %v", err)
			}
			if nextFeedback == nil {
				t.Fatal("expected feedback next page response")
			}
			assertASCLink(t, nextFeedback.Links.Self)
			assertASCLink(t, nextFeedback.Links.Next)
		}

		if len(feedback.Data) == 0 {
			t.Skip("no feedback data to validate filters")
		}

		first := feedback.Data[0].Attributes
		if first.DeviceModel != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetFeedback(
				filteredCtx,
				appID,
				WithFeedbackDeviceModels([]string{first.DeviceModel}),
				WithFeedbackLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered feedback by device model: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no feedback results for device model filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.DeviceModel != first.DeviceModel {
					t.Fatalf("expected device model %q, got %q", first.DeviceModel, item.Attributes.DeviceModel)
				}
			}
		}

		if first.OSVersion != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetFeedback(
				filteredCtx,
				appID,
				WithFeedbackOSVersions([]string{first.OSVersion}),
				WithFeedbackLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered feedback by os version: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no feedback results for os version filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.OSVersion != first.OSVersion {
					t.Fatalf("expected os version %q, got %q", first.OSVersion, item.Attributes.OSVersion)
				}
			}
		}

		if first.AppPlatform != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetFeedback(
				filteredCtx,
				appID,
				WithFeedbackAppPlatforms([]string{first.AppPlatform}),
				WithFeedbackLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered feedback by app platform: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no feedback results for app platform filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.AppPlatform != first.AppPlatform {
					t.Fatalf("expected app platform %q, got %q", first.AppPlatform, item.Attributes.AppPlatform)
				}
			}
		}

		if first.DevicePlatform != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetFeedback(
				filteredCtx,
				appID,
				WithFeedbackDevicePlatforms([]string{first.DevicePlatform}),
				WithFeedbackLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered feedback by device platform: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no feedback results for device platform filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.DevicePlatform != first.DevicePlatform {
					t.Fatalf("expected device platform %q, got %q", first.DevicePlatform, item.Attributes.DevicePlatform)
				}
			}
		}

		sortedCtx, sortedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer sortedCancel()
		sorted, err := client.GetFeedback(
			sortedCtx,
			appID,
			WithFeedbackSort("-createdDate"),
			WithFeedbackLimit(5),
		)
		if err != nil {
			t.Fatalf("failed to fetch sorted feedback: %v", err)
		}
		if sorted == nil {
			t.Fatal("expected sorted feedback response")
		}
		assertLimit(t, len(sorted.Data), 5)
		if len(sorted.Data) < 2 {
			t.Skip("not enough feedback data to validate sort")
		}
		feedbackDates := make([]string, 0, len(sorted.Data))
		for _, item := range sorted.Data {
			feedbackDates = append(feedbackDates, item.Attributes.CreatedDate)
		}
		assertSortedByDateDesc(t, feedbackDates)
	})

	t.Run("crashes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		crashes, err := client.GetCrashes(ctx, appID, WithCrashLimit(1))
		if err != nil {
			t.Fatalf("failed to fetch crashes: %v", err)
		}
		if crashes == nil {
			t.Fatal("expected crashes response")
		}
		assertLimit(t, len(crashes.Data), 1)
		assertASCLink(t, crashes.Links.Self)
		assertASCLink(t, crashes.Links.Next)
		if len(crashes.Data) > 0 && crashes.Data[0].Type == "" {
			t.Fatal("expected crash data type to be set")
		}
		if crashes.Links.Next != "" {
			nextCtx, nextCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer nextCancel()
			nextCrashes, err := client.GetCrashes(nextCtx, appID, WithCrashNextURL(crashes.Links.Next))
			if err != nil {
				t.Fatalf("failed to fetch crashes next page: %v", err)
			}
			if nextCrashes == nil {
				t.Fatal("expected crashes next page response")
			}
			assertASCLink(t, nextCrashes.Links.Self)
			assertASCLink(t, nextCrashes.Links.Next)
		}

		if len(crashes.Data) == 0 {
			t.Skip("no crash data to validate filters")
		}

		first := crashes.Data[0].Attributes
		if first.DeviceModel != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetCrashes(
				filteredCtx,
				appID,
				WithCrashDeviceModels([]string{first.DeviceModel}),
				WithCrashLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered crashes by device model: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no crash results for device model filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.DeviceModel != first.DeviceModel {
					t.Fatalf("expected device model %q, got %q", first.DeviceModel, item.Attributes.DeviceModel)
				}
			}
		}

		if first.OSVersion != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetCrashes(
				filteredCtx,
				appID,
				WithCrashOSVersions([]string{first.OSVersion}),
				WithCrashLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered crashes by os version: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no crash results for os version filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.OSVersion != first.OSVersion {
					t.Fatalf("expected os version %q, got %q", first.OSVersion, item.Attributes.OSVersion)
				}
			}
		}

		if first.AppPlatform != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetCrashes(
				filteredCtx,
				appID,
				WithCrashAppPlatforms([]string{first.AppPlatform}),
				WithCrashLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered crashes by app platform: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no crash results for app platform filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.AppPlatform != first.AppPlatform {
					t.Fatalf("expected app platform %q, got %q", first.AppPlatform, item.Attributes.AppPlatform)
				}
			}
		}

		if first.DevicePlatform != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetCrashes(
				filteredCtx,
				appID,
				WithCrashDevicePlatforms([]string{first.DevicePlatform}),
				WithCrashLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered crashes by device platform: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no crash results for device platform filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.DevicePlatform != first.DevicePlatform {
					t.Fatalf("expected device platform %q, got %q", first.DevicePlatform, item.Attributes.DevicePlatform)
				}
			}
		}

		sortedCtx, sortedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer sortedCancel()
		sorted, err := client.GetCrashes(
			sortedCtx,
			appID,
			WithCrashSort("-createdDate"),
			WithCrashLimit(5),
		)
		if err != nil {
			t.Fatalf("failed to fetch sorted crashes: %v", err)
		}
		if sorted == nil {
			t.Fatal("expected sorted crashes response")
		}
		assertLimit(t, len(sorted.Data), 5)
		if len(sorted.Data) < 2 {
			t.Skip("not enough crash data to validate sort")
		}
		crashDates := make([]string, 0, len(sorted.Data))
		for _, item := range sorted.Data {
			crashDates = append(crashDates, item.Attributes.CreatedDate)
		}
		assertSortedByDateDesc(t, crashDates)
	})

	t.Run("reviews", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		reviews, err := client.GetReviews(ctx, appID, WithLimit(1))
		if err != nil {
			t.Fatalf("failed to fetch reviews: %v", err)
		}
		if reviews == nil {
			t.Fatal("expected reviews response")
		}
		assertLimit(t, len(reviews.Data), 1)
		assertASCLink(t, reviews.Links.Self)
		assertASCLink(t, reviews.Links.Next)
		if len(reviews.Data) > 0 && reviews.Data[0].Type == "" {
			t.Fatal("expected review data type to be set")
		}
		if reviews.Links.Next != "" {
			nextCtx, nextCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer nextCancel()
			nextReviews, err := client.GetReviews(nextCtx, appID, WithNextURL(reviews.Links.Next))
			if err != nil {
				t.Fatalf("failed to fetch reviews next page: %v", err)
			}
			if nextReviews == nil {
				t.Fatal("expected reviews next page response")
			}
			assertASCLink(t, nextReviews.Links.Self)
			assertASCLink(t, nextReviews.Links.Next)
		}

		if len(reviews.Data) == 0 {
			t.Skip("no review data to validate filters")
		}

		first := reviews.Data[0].Attributes
		if first.Rating >= 1 && first.Rating <= 5 {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetReviews(
				filteredCtx,
				appID,
				WithRating(first.Rating),
				WithLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered reviews by rating: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no review results for rating filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.Rating != first.Rating {
					t.Fatalf("expected rating %d, got %d", first.Rating, item.Attributes.Rating)
				}
			}
		}

		if first.Territory != "" {
			filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer filteredCancel()
			filtered, err := client.GetReviews(
				filteredCtx,
				appID,
				WithTerritory(first.Territory),
				WithLimit(5),
			)
			if err != nil {
				t.Fatalf("failed to fetch filtered reviews by territory: %v", err)
			}
			assertLimit(t, len(filtered.Data), 5)
			if len(filtered.Data) == 0 {
				t.Skip("no review results for territory filter")
			}
			for _, item := range filtered.Data {
				if item.Attributes.Territory != first.Territory {
					t.Fatalf("expected territory %q, got %q", first.Territory, item.Attributes.Territory)
				}
			}
		}

		sortedCtx, sortedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer sortedCancel()
		sorted, err := client.GetReviews(
			sortedCtx,
			appID,
			WithReviewSort("-createdDate"),
			WithLimit(5),
		)
		if err != nil {
			t.Fatalf("failed to fetch sorted reviews: %v", err)
		}
		if sorted == nil {
			t.Fatal("expected sorted reviews response")
		}
		assertLimit(t, len(sorted.Data), 5)
		if len(sorted.Data) < 2 {
			t.Skip("not enough review data to validate sort")
		}
		reviewDates := make([]string, 0, len(sorted.Data))
		for _, item := range sorted.Data {
			reviewDates = append(reviewDates, item.Attributes.CreatedDate)
		}
		assertSortedByDateDesc(t, reviewDates)
	})

	t.Run("builds", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		builds, err := client.GetBuilds(ctx, appID, WithBuildsLimit(5))
		if err != nil {
			t.Fatalf("failed to fetch builds: %v", err)
		}
		if builds == nil {
			t.Fatal("expected builds response")
		}
		assertLimit(t, len(builds.Data), 5)
		assertASCLink(t, builds.Links.Self)
		assertASCLink(t, builds.Links.Next)
		if len(builds.Data) == 0 {
			t.Skip("no build data to validate details")
		}
		first := builds.Data[0]
		if first.ID == "" {
			t.Fatal("expected build ID to be set")
		}

		infoCtx, infoCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer infoCancel()
		build, err := client.GetBuild(infoCtx, first.ID)
		if err != nil {
			t.Fatalf("failed to fetch build info: %v", err)
		}
		if build == nil {
			t.Fatal("expected build info response")
		}
		if build.Data.ID != first.ID {
			t.Fatalf("expected build ID %q, got %q", first.ID, build.Data.ID)
		}
		if build.Data.Attributes.Version == "" {
			t.Fatal("expected build version to be set")
		}

		sortedCtx, sortedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer sortedCancel()
		sorted, err := client.GetBuilds(
			sortedCtx,
			appID,
			WithBuildsSort("-uploadedDate"),
			WithBuildsLimit(5),
		)
		if err != nil {
			t.Fatalf("failed to fetch sorted builds: %v", err)
		}
		if sorted == nil {
			t.Fatal("expected sorted builds response")
		}
		assertLimit(t, len(sorted.Data), 5)
		if len(sorted.Data) < 2 {
			t.Skip("not enough builds to validate sort")
		}
		uploadedDates := make([]string, 0, len(sorted.Data))
		for _, item := range sorted.Data {
			uploadedDates = append(uploadedDates, item.Attributes.UploadedDate)
		}
		assertSortedByDateDesc(t, uploadedDates)

		// Note: ExpireBuild is a destructive operation that cannot be undone.
		// Only run this test with ASC_EXPIRE_BUILD_ID set to a build you want to expire.
		if expireID := os.Getenv("ASC_EXPIRE_BUILD_ID"); expireID != "" {
			expireCtx, expireCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer expireCancel()
			expired, err := client.ExpireBuild(expireCtx, expireID)
			if err != nil {
				t.Fatalf("failed to expire build: %v", err)
			}
			if expired == nil {
				t.Fatal("expected expire build response")
			}
			if !expired.Data.Attributes.Expired {
				t.Fatalf("expected build %q to be expired", expireID)
			}
		}
	})
}

func TestIntegrationDevicesReadOnly(t *testing.T) {
	keyID := os.Getenv("ASC_KEY_ID")
	issuerID := os.Getenv("ASC_ISSUER_ID")
	keyPath := os.Getenv("ASC_PRIVATE_KEY_PATH")

	if keyID == "" || issuerID == "" || keyPath == "" {
		t.Skip("integration tests require ASC_KEY_ID, ASC_ISSUER_ID, ASC_PRIVATE_KEY_PATH")
	}

	client, err := NewClient(keyID, issuerID, keyPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	t.Run("devices", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		devices, err := client.GetDevices(ctx, WithDevicesLimit(5))
		if err != nil {
			t.Fatalf("failed to fetch devices: %v", err)
		}
		if devices == nil {
			t.Fatal("expected devices response")
		}
		assertLimit(t, len(devices.Data), 5)
		assertASCLink(t, devices.Links.Self)
		assertASCLink(t, devices.Links.Next)

		if devices.Links.Next != "" {
			nextCtx, nextCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer nextCancel()
			nextDevices, err := client.GetDevices(nextCtx, WithDevicesNextURL(devices.Links.Next))
			if err != nil {
				t.Fatalf("failed to fetch devices next page: %v", err)
			}
			if nextDevices == nil {
				t.Fatal("expected devices next page response")
			}
			assertASCLink(t, nextDevices.Links.Self)
			assertASCLink(t, nextDevices.Links.Next)
		}

		if len(devices.Data) == 0 {
			t.Skip("no devices available to validate filters")
		}

		first := devices.Data[0]
		if first.ID == "" {
			t.Fatal("expected device ID to be set")
		}
		if first.Attributes.UDID == "" {
			t.Skip("device UDID is missing; cannot validate UDID filter")
		}

		filteredCtx, filteredCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer filteredCancel()
		filtered, err := client.GetDevices(
			filteredCtx,
			WithDevicesUDIDs([]string{first.Attributes.UDID}),
			WithDevicesLimit(5),
		)
		if err != nil {
			t.Fatalf("failed to fetch filtered devices by UDID: %v", err)
		}
		assertLimit(t, len(filtered.Data), 5)
		if len(filtered.Data) == 0 {
			t.Skip("no devices returned for UDID filter")
		}
		for _, item := range filtered.Data {
			if item.Attributes.UDID != first.Attributes.UDID {
				t.Fatalf("expected UDID %q, got %q", first.Attributes.UDID, item.Attributes.UDID)
			}
		}

		fieldsCtx, fieldsCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer fieldsCancel()
		_, err = client.GetDevices(fieldsCtx, WithDevicesFields([]string{"name", "udid", "platform", "status"}), WithDevicesLimit(1))
		if err != nil {
			t.Fatalf("failed to fetch devices with fields: %v", err)
		}

		getCtx, getCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer getCancel()
		device, err := client.GetDevice(getCtx, first.ID, []string{"name", "udid", "platform", "status"})
		if err != nil {
			t.Fatalf("failed to fetch device: %v", err)
		}
		if device == nil {
			t.Fatal("expected device response")
		}
		if device.Data.ID != first.ID {
			t.Fatalf("expected device ID %q, got %q", first.ID, device.Data.ID)
		}
	})
}

// TestIntegrationErrorHandling tests API error responses for invalid inputs.
func TestIntegrationErrorHandling(t *testing.T) {
	keyID := os.Getenv("ASC_KEY_ID")
	issuerID := os.Getenv("ASC_ISSUER_ID")
	keyPath := os.Getenv("ASC_PRIVATE_KEY_PATH")

	if keyID == "" || issuerID == "" || keyPath == "" {
		t.Skip("integration tests require ASC_KEY_ID, ASC_ISSUER_ID, ASC_PRIVATE_KEY_PATH")
	}

	client, err := NewClient(keyID, issuerID, keyPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	t.Run("invalid_app_id_feedback", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.GetFeedback(ctx, "invalid-app-id-12345")
		if err == nil {
			t.Fatal("expected error for invalid app ID, got nil")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("invalid_app_id_builds", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.GetBuilds(ctx, "invalid-app-id-12345")
		if err == nil {
			t.Fatal("expected error for invalid app ID, got nil")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("invalid_build_id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.GetBuild(ctx, "invalid-build-id-12345")
		if err == nil {
			t.Fatal("expected error for invalid build ID, got nil")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("invalid_app_id_reviews", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.GetReviews(ctx, "invalid-app-id-12345")
		if err == nil {
			t.Fatal("expected error for invalid app ID, got nil")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("invalid_app_id_crashes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.GetCrashes(ctx, "invalid-app-id-12345")
		if err == nil {
			t.Fatal("expected error for invalid app ID, got nil")
		}
		t.Logf("got expected error: %v", err)
	})
}

func assertLimit(t *testing.T, count, limit int) {
	t.Helper()
	if limit <= 0 {
		return
	}
	if count > limit {
		t.Fatalf("expected at most %d items, got %d", limit, count)
	}
}

func assertASCLink(t *testing.T, link string) {
	t.Helper()
	if link == "" {
		return
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("expected link to be a valid URL, got %q: %v", link, err)
	}
	if parsed.Host != "" && parsed.Host != "api.appstoreconnect.apple.com" {
		t.Fatalf("expected App Store Connect host, got %q", parsed.Host)
	}
	if parsed.Scheme != "" && parsed.Scheme != "https" {
		t.Fatalf("expected https scheme, got %q", parsed.Scheme)
	}
}

func selectLatestAppStoreVersionForTest(versions []Resource[AppStoreVersionAttributes]) Resource[AppStoreVersionAttributes] {
	sort.SliceStable(versions, func(i, j int) bool {
		return parseAppStoreVersionCreatedDateForTest(versions[i]).After(parseAppStoreVersionCreatedDateForTest(versions[j]))
	})
	return versions[0]
}

func parseAppStoreVersionCreatedDateForTest(version Resource[AppStoreVersionAttributes]) time.Time {
	created := strings.TrimSpace(version.Attributes.CreatedDate)
	if created == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, created); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339Nano, created); err == nil {
		return parsed
	}
	return time.Time{}
}

// assertSortedByDateDesc verifies dates are in descending order.
// Works for any date field (createdDate, uploadedDate, etc.)
func assertSortedByDateDesc(t *testing.T, values []string) {
	t.Helper()
	if len(values) < 2 {
		return
	}
	parsed := make([]time.Time, 0, len(values))
	for _, value := range values {
		parsed = append(parsed, parseASCDate(t, value))
	}
	for i := 0; i < len(parsed)-1; i++ {
		if parsed[i].Before(parsed[i+1]) {
			t.Fatalf("expected dates in descending order, got %s before %s", parsed[i], parsed[i+1])
		}
	}
}

func parseASCDate(t *testing.T, value string) time.Time {
	t.Helper()
	if value == "" {
		t.Fatal("expected createdDate to be set")
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	t.Fatalf("failed to parse createdDate %q", value)
	return time.Time{}
}
