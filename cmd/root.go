package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/accessibility"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/actors"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/agerating"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/analytics"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/apps"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/assets"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/auth"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/buildbundles"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/buildlocalizations"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/builds"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/bundleids"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/categories"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/certificates"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/crashes"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/devices"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/encryption"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/eula"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/feedback"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/finance"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/gamecenter"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/iap"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/localizations"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/migrate"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/nominations"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/offercodes"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/performance"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/preorders"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/prerelease"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/pricing"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/profiles"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/publish"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/reviews"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/routingcoverage"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/sandbox"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/signing"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/submit"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/subscriptions"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/testflight"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/users"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/versions"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/xcodecloud"
)

// VersionCommand returns a version subcommand
func VersionCommand(version string) *ffcli.Command {
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "asc version",
		ShortHelp:  "Print version information and exit.",
		UsageFunc:  DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Println(version)
			return nil
		},
	}
}

// RootCommand returns the root command
func RootCommand(version string) *ffcli.Command {
	root := &ffcli.Command{
		Name:       "asc",
		ShortUsage: "asc <subcommand> [flags]",
		ShortHelp:  "A fast, AI-agent friendly CLI for App Store Connect.",
		LongHelp:   "ASC is a lightweight CLI for App Store Connect. Built for developers and AI agents.",
		FlagSet:    flag.NewFlagSet("asc", flag.ExitOnError),
		UsageFunc:  DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			auth.AuthCommand(),
			feedback.FeedbackCommand(),
			crashes.CrashesCommand(),
			reviews.ReviewsCommand(),
			reviews.ReviewCommand(),
			analytics.AnalyticsCommand(),
			performance.PerformanceCommand(),
			finance.FinanceCommand(),
			apps.AppsCommand(),
			apps.AppSetupCommand(),
			apps.AppTagsCommand(),
			nominations.NominationsCommand(),
			bundleids.BundleIDsCommand(),
			certificates.CertificatesCommand(),
			profiles.ProfilesCommand(),
			offercodes.OfferCodesCommand(),
			users.UsersCommand(),
			actors.ActorsCommand(),
			devices.DevicesCommand(),
			testflight.TestFlightCommand(),
			builds.BuildsCommand(),
			buildbundles.BuildBundlesCommand(),
			publish.PublishCommand(),
			versions.VersionsCommand(),
			routingcoverage.RoutingCoverageCommand(),
			apps.AppInfoCommand(),
			eula.EULACommand(),
			pricing.PricingCommand(),
			preorders.PreOrdersCommand(),
			prerelease.PreReleaseVersionsCommand(),
			localizations.LocalizationsCommand(),
			assets.AssetsCommand(),
			buildlocalizations.BuildLocalizationsCommand(),
			testflight.BetaGroupsCommand(),
			testflight.BetaTestersCommand(),
			sandbox.SandboxCommand(),
			signing.SigningCommand(),
			iap.IAPCommand(),
			subscriptions.SubscriptionsCommand(),
			submit.SubmitCommand(),
			xcodecloud.XcodeCloudCommand(),
			categories.CategoriesCommand(),
			agerating.AgeRatingCommand(),
			accessibility.AccessibilityCommand(),
			encryption.EncryptionCommand(),
			migrate.MigrateCommand(),
			gamecenter.GameCenterCommand(),
			VersionCommand(version),
		},
	}

	versionFlag := root.FlagSet.Bool("version", false, "Print version and exit")
	shared.BindRootFlags(root.FlagSet)

	root.Exec = func(ctx context.Context, args []string) error {
		if *versionFlag {
			fmt.Fprintln(os.Stdout, version)
			return nil
		}
		if len(args) > 0 {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
		}
		return flag.ErrHelp
	}

	return root
}
