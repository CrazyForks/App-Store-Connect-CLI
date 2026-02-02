package install

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestInstallSkillsRunsNpxAddSkill(t *testing.T) {
	originalLookup := lookupNpx
	originalRun := runCommand
	t.Cleanup(func() {
		lookupNpx = originalLookup
		runCommand = originalRun
	})

	lookupNpx = func(name string) (string, error) {
		if name != "npx" {
			t.Fatalf("expected lookup for npx, got %q", name)
		}
		return "/bin/npx", nil
	}

	var gotName string
	var gotArgs []string
	runCommand = func(ctx context.Context, name string, args ...string) error {
		gotName = name
		gotArgs = append([]string{}, args...)
		return nil
	}

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}

	if gotName != "/bin/npx" {
		t.Fatalf("expected npx path /bin/npx, got %q", gotName)
	}
	expected := []string{"--yes", "add-skill", defaultSkillsPackage}
	if !reflect.DeepEqual(gotArgs, expected) {
		t.Fatalf("expected args %v, got %v", expected, gotArgs)
	}
}

func TestInstallSkillsAllowsPackageOverride(t *testing.T) {
	originalLookup := lookupNpx
	originalRun := runCommand
	t.Cleanup(func() {
		lookupNpx = originalLookup
		runCommand = originalRun
	})

	lookupNpx = func(name string) (string, error) {
		return "/bin/npx", nil
	}

	var gotArgs []string
	runCommand = func(ctx context.Context, name string, args ...string) error {
		gotArgs = append([]string{}, args...)
		return nil
	}

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{"--package", "example/skills"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}

	expected := []string{"--yes", "add-skill", "example/skills"}
	if !reflect.DeepEqual(gotArgs, expected) {
		t.Fatalf("expected args %v, got %v", expected, gotArgs)
	}
}

func TestInstallSkillsFailsWhenNpxMissing(t *testing.T) {
	originalLookup := lookupNpx
	originalRun := runCommand
	t.Cleanup(func() {
		lookupNpx = originalLookup
		runCommand = originalRun
	})

	lookupNpx = func(name string) (string, error) {
		return "", errors.New("missing")
	}
	runCommand = func(ctx context.Context, name string, args ...string) error {
		t.Fatal("runCommand should not be called when npx is missing")
		return nil
	}

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := cmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errNpxNotFound) {
		t.Fatalf("expected npx error, got %q", err.Error())
	}
}

func TestValidatePackageName(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		wantErr bool
	}{
		{
			name:    "valid repo",
			pkg:     "rudrankriyam/asc-skills",
			wantErr: false,
		},
		{
			name:    "valid scoped",
			pkg:     "@scope/skills",
			wantErr: false,
		},
		{
			name:    "valid name",
			pkg:     "skills_pack-1",
			wantErr: false,
		},
		{
			name:    "invalid leading dash",
			pkg:     "-skills",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			pkg:     "skills$bad",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePackageName(test.pkg)
			if test.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
