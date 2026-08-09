package testflight

import (
	"context"
	"testing"
	"time"
)

func TestContextWithBuildGroupMembershipTimeoutUsesBulkDefault(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "")
	t.Setenv("ASC_TIMEOUT_SECONDS", "")

	ctx, cancel := contextWithBuildGroupMembershipTimeout(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) < 4*time.Minute {
		t.Fatalf("expected five-minute bulk timeout, deadline=%v, present=%t", deadline, ok)
	}
}

func TestContextWithBuildGroupMembershipTimeoutHonorsASCOverride(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "90s")
	t.Setenv("ASC_TIMEOUT_SECONDS", "")

	ctx, cancel := contextWithBuildGroupMembershipTimeout(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	remaining := time.Until(deadline)
	if !ok || remaining < 89*time.Second || remaining > 91*time.Second {
		t.Fatalf("expected ASC_TIMEOUT override near 90s, remaining=%v, present=%t", remaining, ok)
	}
}
