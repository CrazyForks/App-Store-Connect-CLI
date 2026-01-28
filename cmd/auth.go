package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/auth"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/config"
)

const authKeysURL = "https://appstoreconnect.apple.com/access/integrations/api"

var (
	loginJWTGenerator    = asc.GenerateJWT
	loginNetworkValidate = validateLoginNetwork
)

// Auth command factory
func AuthCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "auth",
		ShortUsage: "asc auth <subcommand> [flags]",
		ShortHelp:  "Manage App Store Connect API authentication.",
		LongHelp: `Manage App Store Connect API authentication.

Authentication is handled via App Store Connect API keys. Generate keys at:
https://appstoreconnect.apple.com/access/integrations/api

Credentials are stored in the system keychain when available, with a config fallback.
A repo-local ./.asc/config.json (if present) takes precedence.

Credential resolution order:
  1) Selected profile (keychain/config)
  2) Environment variables (fallback for missing fields)

Use --strict-auth or ASC_STRICT_AUTH=1 to fail when sources are mixed.`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AuthInitCommand(),
			AuthLoginCommand(),
			AuthSwitchCommand(),
			AuthLogoutCommand(),
			AuthValidateCommand(),
			AuthDoctorCommand(),
			AuthStatusCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", args[0])
			return flag.ErrHelp
		},
	}
}

// AuthInit command factory
func AuthInitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth init", flag.ExitOnError)

	force := fs.Bool("force", false, "Overwrite existing config.json")
	local := fs.Bool("local", false, "Write config.json to ./.asc in the current repo")
	open := fs.Bool("open", false, "Open the App Store Connect API keys page in your browser")

	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "asc auth init [flags]",
		ShortHelp:  "Create a template config.json for authentication.",
		LongHelp: `Create a template config.json for authentication.

This writes ~/.asc/config.json with empty fields and secure permissions.
Use --local to write ./.asc/config.json in the current repo instead.

Examples:
  asc auth init
  asc auth init --local
  asc auth init --force
  asc auth init --open`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			var path string
			var err error
			if *local {
				path, err = config.LocalPath()
			} else {
				path, err = config.GlobalPath()
			}
			if err != nil {
				return fmt.Errorf("auth init: %w", err)
			}

			if !*force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("auth init: config already exists at %s (use --force to overwrite)", path)
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("auth init: %w", err)
				}
			}

			template := &config.Config{}
			if err := config.SaveAt(path, template); err != nil {
				return fmt.Errorf("auth init: %w", err)
			}

			if *open {
				if err := openURL(authKeysURL); err != nil {
					return fmt.Errorf("auth init: %w", err)
				}
			}

			result := struct {
				ConfigPath string         `json:"config_path"`
				Created    bool           `json:"created"`
				Config     *config.Config `json:"config"`
			}{
				ConfigPath: path,
				Created:    true,
				Config:     template,
			}
			return asc.PrintJSON(result)
		},
	}
}

type authValidateResult struct {
	Profile          string   `json:"profile,omitempty"`
	KeyID            string   `json:"key_id,omitempty"`
	IssuerID         string   `json:"issuer_id,omitempty"`
	PrivateKeyPath   string   `json:"private_key_path,omitempty"`
	Valid            bool     `json:"valid"`
	NetworkRequested bool     `json:"network_requested"`
	NetworkValid     *bool    `json:"network_valid,omitempty"`
	Errors           []string `json:"errors,omitempty"`
}

// AuthValidate command factory
func AuthValidateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth validate", flag.ExitOnError)

	output := fs.String("output", "json", "Output format: json (default)")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	network := fs.Bool("network", false, "Validate credentials with a lightweight API request")

	return &ffcli.Command{
		Name:       "validate",
		ShortUsage: "asc auth validate [flags]",
		ShortHelp:  "Validate stored authentication credentials.",
		LongHelp: `Validate stored authentication credentials.

Checks the resolved credentials (profile/env/config) and validates the private key file.
Add --network to verify credentials against a lightweight API request.

Examples:
  asc auth validate
  asc --profile "Client" auth validate
  asc auth validate --network`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			normalizedOutput := strings.ToLower(strings.TrimSpace(*output))
			if normalizedOutput != "json" {
				return fmt.Errorf("auth validate: unsupported format: %s (only json is supported)", *output)
			}
			result := authValidateResult{
				Profile:          resolveProfileName(),
				NetworkRequested: *network,
			}

			resolved, err := resolveCredentials()
			if err != nil {
				result.Valid = false
				result.Errors = []string{err.Error()}
				if printErr := printOutput(result, normalizedOutput, *pretty); printErr != nil {
					return printErr
				}
				return NewReportedError(fmt.Errorf("auth validate: %w", err))
			}

			result.KeyID = resolved.keyID
			result.IssuerID = resolved.issuerID
			result.PrivateKeyPath = resolved.keyPath

			if err := auth.ValidateKeyFile(resolved.keyPath); err != nil {
				validationErr := fmt.Errorf("invalid private key: %w", err)
				result.Valid = false
				result.Errors = []string{validationErr.Error()}
				if printErr := printOutput(result, normalizedOutput, *pretty); printErr != nil {
					return printErr
				}
				return NewReportedError(fmt.Errorf("auth validate: %w", validationErr))
			}

			privateKey, err := auth.LoadPrivateKey(resolved.keyPath)
			if err != nil {
				validationErr := fmt.Errorf("failed to load private key: %w", err)
				result.Valid = false
				result.Errors = []string{validationErr.Error()}
				if printErr := printOutput(result, normalizedOutput, *pretty); printErr != nil {
					return printErr
				}
				return NewReportedError(fmt.Errorf("auth validate: %w", validationErr))
			}
			if _, err := asc.GenerateJWT(resolved.keyID, resolved.issuerID, privateKey); err != nil {
				validationErr := fmt.Errorf("failed to generate JWT: %w", err)
				result.Valid = false
				result.Errors = []string{validationErr.Error()}
				if printErr := printOutput(result, normalizedOutput, *pretty); printErr != nil {
					return printErr
				}
				return NewReportedError(fmt.Errorf("auth validate: %w", validationErr))
			}

			var validationErr error
			if *network {
				client, err := asc.NewClient(resolved.keyID, resolved.issuerID, resolved.keyPath)
				if err != nil {
					validationErr = err
				} else if _, err := client.GetApps(ctx, asc.WithAppsLimit(1)); err != nil {
					validationErr = fmt.Errorf("network validation failed: %w", err)
				}

				networkValid := validationErr == nil
				result.NetworkValid = &networkValid
				if validationErr != nil {
					result.Errors = []string{validationErr.Error()}
				}
			}

			result.Valid = validationErr == nil
			if err := printOutput(result, normalizedOutput, *pretty); err != nil {
				return err
			}
			if validationErr != nil {
				return NewReportedError(fmt.Errorf("auth validate: %w", validationErr))
			}
			return nil
		},
	}
}

// AuthDoctor command factory
func AuthDoctorCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth doctor", flag.ExitOnError)

	output := fs.String("output", "text", "Output format: text (default), json")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	fix := fs.Bool("fix", false, "Attempt to fix issues where possible")
	confirm := fs.Bool("confirm", false, "Confirm applying fixes")

	return &ffcli.Command{
		Name:       "doctor",
		ShortUsage: "asc auth doctor [flags]",
		ShortHelp:  "Diagnose authentication configuration issues.",
		LongHelp: `Diagnose authentication configuration issues.

Runs a comprehensive health check across keychain availability, config files,
stored profiles, private key files, and environment variables.

Examples:
  asc auth doctor
  asc auth doctor --output json
  asc auth doctor --fix --confirm`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			normalizedOutput := strings.ToLower(strings.TrimSpace(*output))
			if normalizedOutput != "text" && normalizedOutput != "json" {
				return fmt.Errorf("auth doctor: unsupported format: %s", *output)
			}
			if normalizedOutput != "json" && *pretty {
				return fmt.Errorf("--pretty is only valid with JSON output")
			}
			if *fix && !*confirm {
				return fmt.Errorf("auth doctor: --fix requires --confirm")
			}

			report := auth.Doctor(auth.DoctorOptions{Fix: *fix && *confirm})
			if normalizedOutput == "json" {
				if err := printOutput(report, "json", *pretty); err != nil {
					return err
				}
			} else {
				printDoctorReport(report)
			}

			if report.Summary.Errors > 0 {
				return NewReportedError(fmt.Errorf("auth doctor: found %d error(s)", report.Summary.Errors))
			}
			return nil
		},
	}
}

func printDoctorReport(report auth.DoctorReport) {
	fmt.Println("Auth Doctor")
	for _, section := range report.Sections {
		if len(section.Checks) == 0 {
			continue
		}
		fmt.Printf("\n%s:\n", section.Title)
		for _, check := range section.Checks {
			fmt.Printf("  [%s] %s\n", doctorStatusLabel(check.Status), check.Message)
		}
	}
	if len(report.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	if report.Summary.Errors == 0 && report.Summary.Warnings == 0 {
		fmt.Println("\nNo issues found.")
	} else {
		fmt.Printf("\nFound %d warning(s) and %d error(s).\n", report.Summary.Warnings, report.Summary.Errors)
	}
}

func doctorStatusLabel(status auth.DoctorStatus) string {
	switch status {
	case auth.DoctorOK:
		return "OK"
	case auth.DoctorWarn:
		return "WARN"
	case auth.DoctorFail:
		return "FAIL"
	case auth.DoctorInfo:
		return "INFO"
	default:
		return strings.ToUpper(string(status))
	}
}

func validateLoginCredentials(ctx context.Context, keyID, issuerID, keyPath string, network bool) error {
	privateKey, err := auth.LoadPrivateKey(keyPath)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}
	if _, err := loginJWTGenerator(keyID, issuerID, privateKey); err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}
	if network {
		if err := loginNetworkValidate(ctx, keyID, issuerID, keyPath); err != nil {
			return fmt.Errorf("network validation failed: %w", err)
		}
	}
	return nil
}

func validateLoginNetwork(ctx context.Context, keyID, issuerID, keyPath string) error {
	client, err := asc.NewClient(keyID, issuerID, keyPath)
	if err != nil {
		return err
	}
	_, err = client.GetApps(ctx, asc.WithAppsLimit(1))
	return err
}

func loginStorageMessage(bypassKeychain, local bool) (string, error) {
	if bypassKeychain {
		if local {
			path, err := config.LocalPath()
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Storing credentials in config file at %s", path), nil
		}
		path, err := config.GlobalPath()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Storing credentials in config file at %s", path), nil
	}

	keychainAvailable, err := auth.KeychainAvailable()
	if err != nil {
		return "", err
	}
	if keychainAvailable {
		return "Storing credentials in system keychain", nil
	}
	path, err := config.Path()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("System keychain unavailable; storing credentials in config file at %s", path), nil
}

// AuthLogin command factory
func AuthLoginCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth login", flag.ExitOnError)

	name := fs.String("name", "", "Friendly name for this key")
	keyID := fs.String("key-id", "", "App Store Connect API Key ID")
	issuerID := fs.String("issuer-id", "", "App Store Connect Issuer ID")
	keyPath := fs.String("private-key", "", "Path to private key (.p8) file")
	bypassKeychain := fs.Bool("bypass-keychain", false, "Store credentials in config.json instead of keychain")
	local := fs.Bool("local", false, "When bypassing keychain, write to ./.asc/config.json")
	network := fs.Bool("network", false, "Validate credentials with a lightweight API request")
	skipValidation := fs.Bool("skip-validation", false, "Skip JWT and network validation checks")

	return &ffcli.Command{
		Name:       "login",
		ShortUsage: "asc auth login [flags]",
		ShortHelp:  "Register and store App Store Connect API key.",
		LongHelp: `Register and store App Store Connect API key.

This command stores your API credentials in the system keychain when available,
with a local config fallback (restricted permissions). Use --bypass-keychain to
explicitly bypass keychain and write credentials to ~/.asc/config.json instead.
Add --local to write ./.asc/config.json for the current repo.

Examples:
  asc auth login --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8
  asc auth login --bypass-keychain --local --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8
  asc auth login --network --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8
  asc auth login --skip-validation --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8

The private key file path is stored securely. The key content is never saved.`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *local && !*bypassKeychain {
				return fmt.Errorf("auth login: --local requires --bypass-keychain")
			}
			if *name == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				return flag.ErrHelp
			}
			if *keyID == "" {
				fmt.Fprintln(os.Stderr, "Error: --key-id is required")
				return flag.ErrHelp
			}
			if *issuerID == "" {
				fmt.Fprintln(os.Stderr, "Error: --issuer-id is required")
				return flag.ErrHelp
			}
			if *keyPath == "" {
				fmt.Fprintln(os.Stderr, "Error: --private-key is required")
				return flag.ErrHelp
			}
			if *skipValidation && *network {
				return fmt.Errorf("auth login: --skip-validation and --network are mutually exclusive")
			}

			// Validate the key file exists and is parseable
			if err := auth.ValidateKeyFile(*keyPath); err != nil {
				return fmt.Errorf("auth login: invalid private key: %w", err)
			}

			if !*skipValidation {
				if err := validateLoginCredentials(ctx, *keyID, *issuerID, *keyPath, *network); err != nil {
					return fmt.Errorf("auth login: %w", err)
				}
			}

			storageMessage, err := loginStorageMessage(*bypassKeychain, *local)
			if err != nil {
				return fmt.Errorf("auth login: %w", err)
			}
			fmt.Println(storageMessage)

			// Store credentials securely
			if *bypassKeychain {
				if *local {
					path, err := config.LocalPath()
					if err != nil {
						return fmt.Errorf("auth login: %w", err)
					}
					if err := auth.StoreCredentialsConfigAt(*name, *keyID, *issuerID, *keyPath, path); err != nil {
						return fmt.Errorf("auth login: failed to store credentials: %w", err)
					}
				} else {
					if err := auth.StoreCredentialsConfig(*name, *keyID, *issuerID, *keyPath); err != nil {
						return fmt.Errorf("auth login: failed to store credentials: %w", err)
					}
				}
			} else {
				if err := auth.StoreCredentials(*name, *keyID, *issuerID, *keyPath); err != nil {
					return fmt.Errorf("auth login: failed to store credentials: %w", err)
				}
			}

			fmt.Printf("Successfully registered API key '%s'\n", *name)
			return nil
		},
	}
}

// AuthSwitch command factory
func AuthSwitchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth switch", flag.ExitOnError)

	name := fs.String("name", "", "Profile name to set as default")

	return &ffcli.Command{
		Name:       "switch",
		ShortUsage: "asc auth switch --name <profile>",
		ShortHelp:  "Switch the default authentication profile.",
		LongHelp: `Switch the default authentication profile.

This updates the default profile used for keychain or config credentials.

Examples:
  asc auth switch --name "Personal"
  asc auth switch --name "Client"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedName := strings.TrimSpace(*name)
			if trimmedName == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				return flag.ErrHelp
			}

			credentials, err := auth.ListCredentials()
			if err != nil {
				return fmt.Errorf("auth switch: failed to list credentials: %w", err)
			}
			if len(credentials) == 0 {
				return fmt.Errorf("auth switch: no credentials stored")
			}

			found := false
			for _, cred := range credentials {
				if strings.TrimSpace(cred.Name) == trimmedName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("auth switch: profile %q not found", trimmedName)
			}

			if err := auth.SetDefaultCredentials(trimmedName); err != nil {
				return fmt.Errorf("auth switch: %w", err)
			}

			fmt.Printf("Default profile set to '%s'\n", trimmedName)
			return nil
		},
	}
}

// AuthLogout command factory
func AuthLogoutCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth logout", flag.ExitOnError)
	all := fs.Bool("all", false, "Remove all stored credentials (default)")
	name := fs.String("name", "", "Remove a named credential")

	return &ffcli.Command{
		Name:       "logout",
		ShortUsage: "asc auth logout [flags]",
		ShortHelp:  "Remove stored API credentials.",
		LongHelp: `Remove stored API credentials.

Examples:
  asc auth logout
  asc auth logout --all
  asc auth logout --name "MyKey"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedName := strings.TrimSpace(*name)
			if trimmedName == "" && *name != "" {
				return fmt.Errorf("auth logout: --name cannot be blank")
			}
			if trimmedName != "" && *all {
				return fmt.Errorf("auth logout: --all and --name are mutually exclusive")
			}

			if trimmedName != "" {
				if err := auth.RemoveCredentials(trimmedName); err != nil {
					return fmt.Errorf("auth logout: failed to remove credentials: %w", err)
				}
				fmt.Printf("Successfully removed stored credential '%s'\n", trimmedName)
				return nil
			}

			if err := auth.RemoveAllCredentials(); err != nil {
				return fmt.Errorf("auth logout: failed to remove credentials: %w", err)
			}

			fmt.Println("Successfully removed stored credentials")
			return nil
		},
	}
}

// AuthStatus command factory
func AuthStatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth status", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show detailed storage information")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "asc auth status",
		ShortHelp:  "Show current authentication status.",
		LongHelp: `Show current authentication status.

Displays information about stored API keys and which one is currently active.

Examples:
  asc auth status
  asc auth status --verbose`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			credentials, err := auth.ListCredentials()
			if err != nil {
				return fmt.Errorf("auth status: failed to list credentials: %w", err)
			}

			bypassKeychain := auth.ShouldBypassKeychain()
			keychainAvailable, keychainErr := auth.KeychainAvailable()
			configPath, configErr := config.Path()
			storageBackend := "System Keychain"
			storageLocation := "system keychain"
			var warnings []string

			if bypassKeychain {
				storageBackend = "Config File"
				storageLocation = "unknown"
				if configErr == nil {
					storageLocation = configPath
				}
				warnings = append(warnings, "Keychain bypassed via ASC_BYPASS_KEYCHAIN=1.")
			} else if !keychainAvailable {
				storageBackend = "Config File"
				storageLocation = "unknown"
				if configErr == nil {
					storageLocation = configPath
				}
				if keychainErr != nil {
					warnings = append(warnings, fmt.Sprintf("System keychain error: %v", keychainErr))
				} else {
					warnings = append(warnings, "System keychain unavailable. Credentials stored in config file.")
				}
			}

			hasConfigCreds := false
			for _, cred := range credentials {
				if cred.Source == "config" {
					hasConfigCreds = true
					break
				}
			}
			if hasConfigCreds && keychainAvailable && !bypassKeychain {
				warnings = append(warnings, "Some credentials are stored in config file (less secure).")
			}

			fmt.Printf("Credential storage: %s\n", storageBackend)
			fmt.Printf("Location: %s\n", storageLocation)
			for _, warning := range warnings {
				fmt.Printf("Warning: %s\n", warning)
			}
			if *verbose {
				fmt.Printf("Keychain available: %t\n", keychainAvailable)
				if keychainErr != nil {
					fmt.Printf("Keychain error: %v\n", keychainErr)
				}
				if configErr == nil {
					fmt.Printf("Config path: %s\n", configPath)
				}
			}
			fmt.Println()

			if len(credentials) == 0 {
				fmt.Println("No credentials stored. Run 'asc auth login' to get started.")
			} else {
				fmt.Println("Stored credentials:")
				for _, cred := range credentials {
					active := ""
					if cred.IsDefault {
						active = " (default)"
					}
					fmt.Printf("  - %s (Key ID: %s)%s (stored in %s)\n", cred.Name, cred.KeyID, active, credentialStorageLabel(cred))
				}
			}

			profile := resolveProfileName()
			envKeyID := strings.TrimSpace(os.Getenv("ASC_KEY_ID"))
			envIssuerID := strings.TrimSpace(os.Getenv("ASC_ISSUER_ID"))
			hasKeyEnv := strings.TrimSpace(os.Getenv("ASC_PRIVATE_KEY_PATH")) != "" ||
				strings.TrimSpace(os.Getenv(privateKeyEnvVar)) != "" ||
				strings.TrimSpace(os.Getenv(privateKeyBase64EnvVar)) != ""
			envProvided := envKeyID != "" || envIssuerID != "" || hasKeyEnv
			envComplete := envKeyID != "" && envIssuerID != "" && hasKeyEnv

			if profile != "" && envProvided {
				fmt.Printf("Profile %q selected; environment credentials will be ignored.\n", profile)
			} else if bypassKeychain && envComplete {
				fmt.Printf("Environment credentials detected (ASC_KEY_ID: %s). With ASC_BYPASS_KEYCHAIN=1, they will be used when no profile is selected.\n", envKeyID)
			} else if bypassKeychain && envProvided && !envComplete {
				fmt.Println("Environment credentials are incomplete. Set ASC_KEY_ID, ASC_ISSUER_ID, and one of ASC_PRIVATE_KEY_PATH/ASC_PRIVATE_KEY/ASC_PRIVATE_KEY_B64.")
			}
			return nil
		},
	}
}

func credentialStorageLabel(cred auth.Credential) string {
	if strings.TrimSpace(cred.SourcePath) != "" {
		return fmt.Sprintf("%s: %s", cred.Source, cred.SourcePath)
	}
	if strings.TrimSpace(cred.Source) != "" {
		return cred.Source
	}
	return "unknown"
}
