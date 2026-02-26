package web

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeclarationToTupleSetNotCollected(t *testing.T) {
	tuples, err := declarationToTupleSet(privacyDeclarationFile{
		SchemaVersion: privacySchemaVersion,
		DataUsages: []privacyUsage{
			{
				DataProtections: []string{dataProtectionNotCollected},
			},
		},
	})
	if err != nil {
		t.Fatalf("declarationToTupleSet() error = %v", err)
	}
	if len(tuples) != 1 {
		t.Fatalf("expected one tuple, got %d", len(tuples))
	}
	wantKey := privacyTupleKey(privacyTuple{DataProtection: dataProtectionNotCollected})
	if _, ok := tuples[wantKey]; !ok {
		t.Fatalf("expected not-collected tuple key %q, got %#v", wantKey, tuples)
	}
}

func TestDeclarationToTupleSetRejectsCategoryForNotCollected(t *testing.T) {
	_, err := declarationToTupleSet(privacyDeclarationFile{
		SchemaVersion: privacySchemaVersion,
		DataUsages: []privacyUsage{
			{
				Category:        "NAME",
				DataProtections: []string{dataProtectionNotCollected},
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "cannot include category") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeclarationToTupleSetRejectsCollectedWithoutPurpose(t *testing.T) {
	_, err := declarationToTupleSet(privacyDeclarationFile{
		SchemaVersion: privacySchemaVersion,
		DataUsages: []privacyUsage{
			{
				Category:        "NAME",
				DataProtections: []string{dataProtectionLinked},
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "purposes is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeclarationFromTupleSetGroupsByCategoryAndPurpose(t *testing.T) {
	declaration := declarationFromTupleSet(map[string]privacyTuple{
		privacyTupleKey(privacyTuple{
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionLinked,
		}): {
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionLinked,
		},
		privacyTupleKey(privacyTuple{
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionTracking,
		}): {
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionTracking,
		},
	})

	if declaration.SchemaVersion != privacySchemaVersion {
		t.Fatalf("expected schemaVersion=%d, got %d", privacySchemaVersion, declaration.SchemaVersion)
	}
	if len(declaration.DataUsages) != 1 {
		t.Fatalf("expected one usage group, got %d", len(declaration.DataUsages))
	}
	got := declaration.DataUsages[0]
	if got.Category != "NAME" || len(got.Purposes) != 1 || got.Purposes[0] != "APP_FUNCTIONALITY" {
		t.Fatalf("unexpected grouped usage: %#v", got)
	}
	if !reflect.DeepEqual(got.DataProtections, []string{dataProtectionLinked, dataProtectionTracking}) {
		t.Fatalf("unexpected protections: %#v", got.DataProtections)
	}
}

func TestDeclarationFromRemoteDataUsagesEmptyDefaultsNotCollected(t *testing.T) {
	declaration := declarationFromRemoteDataUsages(nil)

	if declaration.SchemaVersion != privacySchemaVersion {
		t.Fatalf("expected schemaVersion=%d, got %d", privacySchemaVersion, declaration.SchemaVersion)
	}
	if len(declaration.DataUsages) != 1 {
		t.Fatalf("expected one default data usage, got %d", len(declaration.DataUsages))
	}
	if !reflect.DeepEqual(declaration.DataUsages[0].DataProtections, []string{dataProtectionNotCollected}) {
		t.Fatalf("unexpected default declaration: %#v", declaration.DataUsages[0])
	}
	if declaration.DataUsages[0].Category != "" || len(declaration.DataUsages[0].Purposes) != 0 {
		t.Fatalf("expected DATA_NOT_COLLECTED declaration with empty category/purposes, got %#v", declaration.DataUsages[0])
	}
}

func TestPlanFromDesiredAndRemoteIncludesDuplicateRemoteDeletes(t *testing.T) {
	desired := map[string]privacyTuple{
		privacyTupleKey(privacyTuple{
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionLinked,
		}): {
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionLinked,
		},
	}
	remote := map[string]privacyRemoteState{
		privacyTupleKey(privacyTuple{
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionLinked,
		}): {
			Tuple: privacyTuple{
				Category:       "NAME",
				Purpose:        "APP_FUNCTIONALITY",
				DataProtection: dataProtectionLinked,
			},
			UsageIDs: []string{"usage-1", "usage-2"},
		},
	}

	plan := planFromDesiredAndRemote("123", "./privacy.json", desired, remote)
	if len(plan.Adds) != 0 {
		t.Fatalf("expected no adds, got %#v", plan.Adds)
	}
	if len(plan.Deletes) != 1 {
		t.Fatalf("expected one duplicate delete, got %#v", plan.Deletes)
	}
	if plan.Deletes[0].UsageID != "usage-2" {
		t.Fatalf("expected usage-2 delete, got %#v", plan.Deletes[0])
	}
}

func TestPlanFromDesiredAndRemoteSkipsDeletesWithoutUsageID(t *testing.T) {
	desired := map[string]privacyTuple{}
	remote := map[string]privacyRemoteState{
		privacyTupleKey(privacyTuple{
			Category:       "NAME",
			Purpose:        "APP_FUNCTIONALITY",
			DataProtection: dataProtectionLinked,
		}): {
			Tuple: privacyTuple{
				Category:       "NAME",
				Purpose:        "APP_FUNCTIONALITY",
				DataProtection: dataProtectionLinked,
			},
			UsageIDs: nil,
		},
	}

	plan := planFromDesiredAndRemote("123", "./privacy.json", desired, remote)
	if len(plan.Deletes) != 0 {
		t.Fatalf("expected no deletes for remote tuples without usage IDs, got %#v", plan.Deletes)
	}
	if len(plan.SkippedDeletes) != 1 {
		t.Fatalf("expected one skipped delete for missing usage id, got %#v", plan.SkippedDeletes)
	}
	if plan.SkippedDeletes[0].Reason != "missing_usage_id" {
		t.Fatalf("expected missing_usage_id reason, got %#v", plan.SkippedDeletes[0])
	}
	if len(plan.APICalls) != 0 {
		t.Fatalf("expected no delete api calls for remote tuples without usage IDs, got %#v", plan.APICalls)
	}
}

func TestParsePrivacyDeclarationFileRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "privacy.json")
	if err := os.WriteFile(path, []byte(`{
		"schemaVersion": 1,
		"dataUsages": [
			{
				"category": "NAME",
				"purposes": ["APP_FUNCTIONALITY"],
				"dataProtections": ["DATA_LINKED_TO_YOU"],
				"unknownField": "x"
			}
		]
	}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := parsePrivacyDeclarationFile(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}
