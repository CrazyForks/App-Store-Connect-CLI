//go:build darwin

package xcode

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	legacyexportoptions "github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/exportoptionsgenerator"
	"github.com/bitrise-io/go-xcode/v2/xcarchive"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"
)

// buildPlatformExportOptionsPayload uses Bitrise's current typed v2 model on
// macOS, where xcodebuild and local signing asset resolution are available.
func buildPlatformExportOptionsPayload(opts ExportOptionsGenerateOptions, teamID string, manual manualExportOptions) map[string]any {
	model := exportoptions.NewAppStoreConnectOptions(exportoptions.MethodAppStoreConnect)
	model.TeamID = teamID
	model.Destination = exportoptions.Destination(opts.Destination)
	model.SigningStyle = exportoptions.SigningStyle(opts.SigningStyle)
	if opts.SigningStyle == exportOptionsSigningStyleManual {
		model.SigningCertificate = manual.SigningCertificate
		model.BundleIDProvisioningProfileMapping = cloneProvisioningProfiles(manual.ProvisioningProfiles)
	}
	return model.Hash()
}

func generateManualExportOptions(ctx context.Context, archivePath, teamID string) (manualExportOptions, error) {
	if err := contextError(ctx); err != nil {
		return manualExportOptions{}, err
	}
	archive, err := xcarchive.NewIosArchive(archivePath)
	if err != nil {
		return manualExportOptions{}, fmt.Errorf("read iOS archive: %w", err)
	}
	archiveInfo, err := exportoptionsgenerator.ReadArchiveExportInfo(archive)
	if err != nil {
		return manualExportOptions{}, fmt.Errorf("read archive export information: %w", err)
	}
	generator := exportoptionsgenerator.New(
		xcodeversion.NewXcodeVersionProvider(command.NewFactory(env.NewRepository())),
		log.NewLogger(),
	)
	generated, err := generator.GenerateApplicationExportOptions(
		exportoptionsgenerator.ExportProductApp,
		archiveInfo,
		// Bitrise v2's generator currently exposes these v1 argument types.
		legacyexportoptions.MethodAppStoreConnect,
		legacyexportoptions.SigningStyleManual,
		exportoptionsgenerator.Opts{TeamID: teamID},
	)
	if err != nil {
		return manualExportOptions{}, err
	}
	return manualExportOptionsFromHash(generated.Hash())
}

func manualExportOptionsFromHash(payload map[string]interface{}) (manualExportOptions, error) {
	profiles, err := provisioningProfilesFromPayload(payload["provisioningProfiles"])
	if err != nil {
		return manualExportOptions{}, err
	}
	return manualExportOptions{
		TeamID:               strings.TrimSpace(coercePlistValueToString(payload["teamID"])),
		SigningCertificate:   strings.TrimSpace(coercePlistValueToString(payload["signingCertificate"])),
		ProvisioningProfiles: profiles,
	}, nil
}
