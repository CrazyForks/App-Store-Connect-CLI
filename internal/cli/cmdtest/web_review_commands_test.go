package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestWebSubmissionsCommandsAreRegistered(t *testing.T) {
	root := RootCommand("1.2.3")
	for _, path := range [][]string{
		{"web", "submissions"},
		{"web", "submissions", "list"},
		{"web", "submissions", "show"},
		{"web", "submissions", "items"},
	} {
		if sub := findSubcommand(root, path...); sub == nil {
			t.Fatalf("expected command %q to be registered", strings.Join(path, " "))
		}
	}
}

func TestWebReviewCommandsAreRegistered(t *testing.T) {
	root := RootCommand("1.2.3")
	for _, path := range [][]string{
		{"web", "review"},
		{"web", "review", "threads", "list"},
		{"web", "review", "messages", "list"},
		{"web", "review", "rejections", "list"},
		{"web", "review", "draft", "show"},
		{"web", "review", "attachments", "list"},
		{"web", "review", "attachments", "download"},
	} {
		if sub := findSubcommand(root, path...); sub == nil {
			t.Fatalf("expected command %q to be registered", strings.Join(path, " "))
		}
	}
}

func TestWebSubmissionsListRequiresApp(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"web", "submissions", "list"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "--app is required") {
		t.Fatalf("expected missing --app message, got %q", stderr)
	}
}

func TestWebSubmissionsListRejectsUnknownState(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"web", "submissions", "list",
			"--app", "123456789",
			"--state", "UNRESOLVED_ISSUES,NOT_A_REAL_STATE",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "unsupported value") {
		t.Fatalf("expected unsupported state message, got %q", stderr)
	}
}

func TestWebSubmissionsListRequiresAppleIDWhenNoMatchingCache(t *testing.T) {
	t.Setenv("ASC_WEB_SESSION_CACHE_BACKEND", "file")
	t.Setenv("ASC_WEB_SESSION_CACHE_DIR", t.TempDir())
	t.Setenv("ASC_WEB_SESSION_CACHE", "1")
	t.Setenv("ASC_WEB_PASSWORD", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"web", "submissions", "list",
			"--app", "123456789",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "--apple-id is required") {
		t.Fatalf("expected missing apple-id message, got %q", stderr)
	}
}

func TestWebReviewThreadsListRequiresSingleSelector(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	t.Run("missing both", func(t *testing.T) {
		var runErr error
		_, stderr := captureOutput(t, func() {
			if err := root.Parse([]string{"web", "review", "threads", "list"}); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			runErr = root.Run(context.Background())
		})
		if !errors.Is(runErr, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", runErr)
		}
		if !strings.Contains(stderr, "exactly one of --app or --submission is required") {
			t.Fatalf("unexpected stderr: %q", stderr)
		}
	})

	t.Run("both provided", func(t *testing.T) {
		var runErr error
		_, stderr := captureOutput(t, func() {
			if err := root.Parse([]string{
				"web", "review", "threads", "list",
				"--app", "123",
				"--submission", "456",
			}); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			runErr = root.Run(context.Background())
		})
		if !errors.Is(runErr, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", runErr)
		}
		if !strings.Contains(stderr, "exactly one of --app or --submission is required") {
			t.Fatalf("unexpected stderr: %q", stderr)
		}
	})
}

func TestWebReviewAttachmentsListRequiresSingleSelector(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"web", "review", "attachments", "list"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "exactly one of --thread or --submission is required") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestWebReviewAttachmentsDownloadRequiresOutDir(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"web", "review", "attachments", "download",
			"--thread", "111",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "--out is required") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}
