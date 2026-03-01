package web

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
	webcore "github.com/rudrankriyam/App-Store-Connect-CLI/internal/web"
)

func webXcodeCloudEnvVarsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud env-vars", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "env-vars",
		ShortUsage: "asc web xcode-cloud env-vars <subcommand> [flags]",
		ShortHelp:  "EXPERIMENTAL: Manage Xcode Cloud workflow environment variables.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

List, set, and delete environment variables on Xcode Cloud workflows
using Apple's private CI API. Requires a web session.

` + webWarningText + `

Examples:
  asc web xcode-cloud env-vars list --product-id "UUID" --workflow-id "WF-UUID" --apple-id "user@example.com"
  asc web xcode-cloud env-vars set --product-id "UUID" --workflow-id "WF-UUID" --name MY_VAR --value hello --apple-id "user@example.com"
  asc web xcode-cloud env-vars set --product-id "UUID" --workflow-id "WF-UUID" --name MY_SECRET --value s3cret --secret --apple-id "user@example.com"
  asc web xcode-cloud env-vars delete --product-id "UUID" --workflow-id "WF-UUID" --name MY_VAR --apple-id "user@example.com"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			webXcodeCloudEnvVarsListCommand(),
			webXcodeCloudEnvVarsSetCommand(),
			webXcodeCloudEnvVarsDeleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// CIEnvVarsListResult is the output type for the env-vars list command.
type CIEnvVarsListResult struct {
	WorkflowID string                          `json:"workflow_id"`
	Variables  []webcore.CIEnvironmentVariable `json:"variables"`
}

func webXcodeCloudEnvVarsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud env-vars list", flag.ExitOnError)
	sessionFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	productID := fs.String("product-id", "", "Xcode Cloud product ID (required)")
	workflowID := fs.String("workflow-id", "", "Xcode Cloud workflow ID (required)")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc web xcode-cloud env-vars list --product-id ID --workflow-id ID [flags]",
		ShortHelp:  "EXPERIMENTAL: List workflow environment variables.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

List environment variables for an Xcode Cloud workflow.
Plaintext variables show their values; secret variables show "(redacted)".

` + webWarningText + `

Examples:
  asc web xcode-cloud env-vars list --product-id "UUID" --workflow-id "WF-UUID" --apple-id "user@example.com"
  asc web xcode-cloud env-vars list --product-id "UUID" --workflow-id "WF-UUID" --apple-id "user@example.com" --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			pid := strings.TrimSpace(*productID)
			if pid == "" {
				fmt.Fprintln(os.Stderr, "Error: --product-id is required")
				return flag.ErrHelp
			}
			wfID := strings.TrimSpace(*workflowID)
			if wfID == "" {
				fmt.Fprintln(os.Stderr, "Error: --workflow-id is required")
				return flag.ErrHelp
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud env-vars list failed: session has no public provider ID")
			}

			client := newCIClientFn(session)
			workflow, err := client.GetCIWorkflow(requestCtx, teamID, pid, wfID)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud env-vars list")
			}
			vars, err := webcore.ExtractEnvVars(workflow.Content)
			if err != nil {
				return fmt.Errorf("xcode-cloud env-vars list failed: %w", err)
			}

			result := &CIEnvVarsListResult{
				WorkflowID: wfID,
				Variables:  vars,
			}
			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderEnvVarsTable(result) },
				func() error { return renderEnvVarsMarkdown(result) },
			)
		},
	}
}

func webXcodeCloudEnvVarsSetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud env-vars set", flag.ExitOnError)
	sessionFlags := bindWebSessionFlags(fs)

	productID := fs.String("product-id", "", "Xcode Cloud product ID (required)")
	workflowID := fs.String("workflow-id", "", "Xcode Cloud workflow ID (required)")
	name := fs.String("name", "", "Environment variable name (required)")
	value := fs.String("value", "", "Environment variable value (required)")
	secret := fs.Bool("secret", false, "Encrypt the value as a secret")

	return &ffcli.Command{
		Name:       "set",
		ShortUsage: "asc web xcode-cloud env-vars set --product-id ID --workflow-id ID --name NAME --value VALUE [--secret] [flags]",
		ShortHelp:  "EXPERIMENTAL: Set a workflow environment variable.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Set (create or update) an environment variable on an Xcode Cloud workflow.
Use --secret to encrypt the value using ECIES (the same scheme as the ASC web UI).
If a variable with the same name already exists, it will be updated.

` + webWarningText + `

Examples:
  asc web xcode-cloud env-vars set --product-id "UUID" --workflow-id "WF-UUID" --name MY_VAR --value hello --apple-id "user@example.com"
  asc web xcode-cloud env-vars set --product-id "UUID" --workflow-id "WF-UUID" --name MY_SECRET --value s3cret --secret --apple-id "user@example.com"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			pid := strings.TrimSpace(*productID)
			if pid == "" {
				fmt.Fprintln(os.Stderr, "Error: --product-id is required")
				return flag.ErrHelp
			}
			wfID := strings.TrimSpace(*workflowID)
			if wfID == "" {
				fmt.Fprintln(os.Stderr, "Error: --workflow-id is required")
				return flag.ErrHelp
			}
			varName := strings.TrimSpace(*name)
			if varName == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				return flag.ErrHelp
			}
			varValue := *value
			if varValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --value is required")
				return flag.ErrHelp
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud env-vars set failed: session has no public provider ID")
			}

			client := newCIClientFn(session)

			// Get current workflow
			workflow, err := client.GetCIWorkflow(requestCtx, teamID, pid, wfID)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud env-vars set")
			}
			vars, err := webcore.ExtractEnvVars(workflow.Content)
			if err != nil {
				return fmt.Errorf("xcode-cloud env-vars set failed: %w", err)
			}

			// Build the new env var
			var envVar webcore.CIEnvironmentVariable
			envVar.Name = varName

			if *secret {
				// Fetch encryption key and encrypt
				keyResp, err := client.GetCIEncryptionKey(requestCtx)
				if err != nil {
					return fmt.Errorf("xcode-cloud env-vars set failed: could not fetch encryption key: %w", err)
				}
				ct, err := webcore.ECIESEncrypt(keyResp.Key, varValue)
				if err != nil {
					return fmt.Errorf("xcode-cloud env-vars set failed: encryption error: %w", err)
				}
				envVar.Value = webcore.CIEnvironmentVariableValue{Ciphertext: &ct}
			} else {
				envVar.Value = webcore.CIEnvironmentVariableValue{Plaintext: &varValue}
			}

			// Upsert: find existing by name or append
			found := false
			for i, v := range vars {
				if strings.EqualFold(v.Name, varName) {
					envVar.ID = v.ID
					vars[i] = envVar
					found = true
					break
				}
			}
			if !found {
				envVar.ID = newUUID()
				vars = append(vars, envVar)
			}

			// Update workflow
			newContent, err := webcore.SetEnvVars(workflow.Content, vars)
			if err != nil {
				return fmt.Errorf("xcode-cloud env-vars set failed: %w", err)
			}
			if err := client.UpdateCIWorkflow(requestCtx, teamID, pid, wfID, newContent); err != nil {
				return withWebAuthHint(err, "xcode-cloud env-vars set")
			}

			varType := "plaintext"
			if *secret {
				varType = "secret"
			}
			wfName := extractWorkflowName(workflow.Content)
			fmt.Fprintf(os.Stdout, "Set %s environment variable %q on workflow %s (%s)\n", varType, varName, wfName, wfID)
			return nil
		},
	}
}

func webXcodeCloudEnvVarsDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud env-vars delete", flag.ExitOnError)
	sessionFlags := bindWebSessionFlags(fs)

	productID := fs.String("product-id", "", "Xcode Cloud product ID (required)")
	workflowID := fs.String("workflow-id", "", "Xcode Cloud workflow ID (required)")
	name := fs.String("name", "", "Environment variable name to delete (required)")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "asc web xcode-cloud env-vars delete --product-id ID --workflow-id ID --name NAME [flags]",
		ShortHelp:  "EXPERIMENTAL: Delete a workflow environment variable.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Delete an environment variable from an Xcode Cloud workflow by name.

` + webWarningText + `

Examples:
  asc web xcode-cloud env-vars delete --product-id "UUID" --workflow-id "WF-UUID" --name MY_VAR --apple-id "user@example.com"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			pid := strings.TrimSpace(*productID)
			if pid == "" {
				fmt.Fprintln(os.Stderr, "Error: --product-id is required")
				return flag.ErrHelp
			}
			wfID := strings.TrimSpace(*workflowID)
			if wfID == "" {
				fmt.Fprintln(os.Stderr, "Error: --workflow-id is required")
				return flag.ErrHelp
			}
			varName := strings.TrimSpace(*name)
			if varName == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				return flag.ErrHelp
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud env-vars delete failed: session has no public provider ID")
			}

			client := newCIClientFn(session)

			// Get current workflow
			workflow, err := client.GetCIWorkflow(requestCtx, teamID, pid, wfID)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud env-vars delete")
			}
			vars, err := webcore.ExtractEnvVars(workflow.Content)
			if err != nil {
				return fmt.Errorf("xcode-cloud env-vars delete failed: %w", err)
			}

			// Find and remove the variable
			found := false
			filtered := make([]webcore.CIEnvironmentVariable, 0, len(vars))
			for _, v := range vars {
				if strings.EqualFold(v.Name, varName) {
					found = true
					continue
				}
				filtered = append(filtered, v)
			}
			if !found {
				return fmt.Errorf("environment variable %q not found in workflow %s", varName, wfID)
			}

			// Update workflow
			newContent, err := webcore.SetEnvVars(workflow.Content, filtered)
			if err != nil {
				return fmt.Errorf("xcode-cloud env-vars delete failed: %w", err)
			}
			if err := client.UpdateCIWorkflow(requestCtx, teamID, pid, wfID, newContent); err != nil {
				return withWebAuthHint(err, "xcode-cloud env-vars delete")
			}

			wfName := extractWorkflowName(workflow.Content)
			fmt.Fprintf(os.Stdout, "Deleted environment variable %q from workflow %s (%s)\n", varName, wfName, wfID)
			return nil
		},
	}
}

func renderEnvVarsTable(result *CIEnvVarsListResult) error {
	if result == nil || len(result.Variables) == 0 {
		fmt.Println("No environment variables found.")
		return nil
	}
	asc.RenderTable(
		[]string{"Name", "Type", "Value"},
		buildEnvVarRows(result.Variables),
	)
	return nil
}

func renderEnvVarsMarkdown(result *CIEnvVarsListResult) error {
	if result == nil || len(result.Variables) == 0 {
		fmt.Println("No environment variables found.")
		return nil
	}
	asc.RenderMarkdown(
		[]string{"Name", "Type", "Value"},
		buildEnvVarRows(result.Variables),
	)
	return nil
}

func buildEnvVarRows(vars []webcore.CIEnvironmentVariable) [][]string {
	rows := make([][]string, 0, len(vars))
	for _, v := range vars {
		varType := "plaintext"
		varValue := ""
		switch {
		case v.Value.Plaintext != nil:
			varType = "plaintext"
			varValue = *v.Value.Plaintext
		case v.Value.Ciphertext != nil || v.Value.RedactedValue != nil:
			varType = "secret"
			varValue = "(redacted)"
		}
		rows = append(rows, []string{v.Name, varType, varValue})
	}
	return rows
}

// extractWorkflowName extracts the "name" field from raw workflow content JSON.
func extractWorkflowName(content json.RawMessage) string {
	var m struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(content, &m); err != nil || m.Name == "" {
		return "unknown"
	}
	return m.Name
}

// newUUID generates a random UUID v4 string.
func newUUID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
