package release

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// verifyResumedCheckpointBinding re-establishes a checkpoint's app, version,
// build, and submission binding from authenticated API state.
//
// The checkpoint file is unsigned local JSON, so matching user-facing arguments
// proves nothing about the IDs and completed-step flags stored alongside them. A
// stored ID that cannot be tied back to the selected app, version string, and
// platform aborts the run. A completed mutation step that current API state
// contradicts is discarded so the step runs again against the verified target
// instead of being reported as done.
func verifyResumedCheckpointBinding(
	ctx context.Context,
	client *asc.Client,
	opts runOptions,
	checkpoint *runCheckpoint,
	emit func(string),
) error {
	if checkpoint == nil {
		return nil
	}
	emitMessage := func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		if emit != nil {
			emit(message)
			return
		}
		fmt.Fprintln(os.Stderr, message)
	}

	for name := range checkpoint.Completed {
		if name == stepSubmitReview && !opts.SubmitForReview {
			return fmt.Errorf("checkpoint completed step %q requires resuming with --submit-for-review", name)
		}
		if !isReleasePipelineStep(name, opts.SubmitForReview) {
			return fmt.Errorf("checkpoint records unknown completed step %q", name)
		}
	}

	versionID := strings.TrimSpace(checkpoint.VersionID)
	if versionID == "" {
		if len(checkpoint.Completed) > 0 {
			return fmt.Errorf("checkpoint reports completed steps without a version ID to verify them against")
		}
		return nil
	}

	version, err := shared.ResolveOwnedAppStoreVersionByID(ctx, client, opts.AppID, versionID, opts.Platform)
	if err != nil {
		return fmt.Errorf("checkpoint version %s could not be verified: %w", versionID, err)
	}
	if resolved := strings.TrimSpace(version.Attributes.VersionString); !strings.EqualFold(resolved, strings.TrimSpace(opts.Version)) {
		return fmt.Errorf("checkpoint version %s is version %q, not %q", versionID, resolved, strings.TrimSpace(opts.Version))
	}

	if checkpoint.Completed[stepAttachBuild] {
		attachedBuildID, buildErr := attachedAppStoreVersionBuildID(ctx, client, versionID)
		switch {
		case buildErr != nil:
			delete(checkpoint.Completed, stepAttachBuild)
			emitMessage("Rechecking %s: could not confirm the build attached to version %s (%v).", stepAttachBuild, versionID, buildErr)
		case attachedBuildID != strings.TrimSpace(opts.BuildID):
			delete(checkpoint.Completed, stepAttachBuild)
			emitMessage(
				"Rechecking %s: version %s currently has build %q attached, not %q.",
				stepAttachBuild,
				versionID,
				attachedBuildID,
				strings.TrimSpace(opts.BuildID),
			)
		}
	}

	if checkpoint.Completed[stepSubmitReview] {
		submissionID := strings.TrimSpace(checkpoint.SubmissionID)
		boundVersionID, submissionState, submissionErr := reviewSubmissionBinding(ctx, client, submissionID)
		switch {
		case submissionID == "":
			delete(checkpoint.Completed, stepSubmitReview)
			emitMessage("Rechecking %s: the checkpoint reports a submission without recording its ID.", stepSubmitReview)
		case submissionErr != nil:
			delete(checkpoint.Completed, stepSubmitReview)
			checkpoint.SubmissionID = ""
			emitMessage("Rechecking %s: could not confirm review submission %s (%v).", stepSubmitReview, submissionID, submissionErr)
		case boundVersionID != versionID:
			delete(checkpoint.Completed, stepSubmitReview)
			checkpoint.SubmissionID = ""
			emitMessage(
				"Rechecking %s: review submission %s is bound to version %q, not %q.",
				stepSubmitReview,
				submissionID,
				boundVersionID,
				versionID,
			)
		case !reviewSubmissionStateProvesSubmission(submissionState):
			delete(checkpoint.Completed, stepSubmitReview)
			checkpoint.SubmissionID = ""
			emitMessage(
				"Rechecking %s: review submission %s is in state %q, which does not prove it was submitted.",
				stepSubmitReview,
				submissionID,
				string(submissionState),
			)
		}
	}

	return nil
}

func isReleasePipelineStep(name string, submitForReview bool) bool {
	switch name {
	case stepEnsureVersion, stepApplyMetadata, stepAttachBuild, stepValidateReadiness:
		return true
	case stepSubmitReview:
		return submitForReview
	default:
		return false
	}
}

func attachedAppStoreVersionBuildID(ctx context.Context, client *asc.Client, versionID string) (string, error) {
	resp, err := client.GetAppStoreVersionBuild(ctx, versionID)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("empty build response for version %s", versionID)
	}
	return strings.TrimSpace(resp.Data.ID), nil
}

func reviewSubmissionBinding(ctx context.Context, client *asc.Client, submissionID string) (string, asc.ReviewSubmissionState, error) {
	if strings.TrimSpace(submissionID) == "" {
		return "", "", nil
	}
	resp, err := client.GetReviewSubmission(ctx, submissionID)
	if err != nil {
		return "", "", err
	}
	if resp == nil {
		return "", "", nil
	}
	state := resp.Data.Attributes.SubmissionState
	if resp.Data.Relationships == nil || resp.Data.Relationships.AppStoreVersionForReview == nil {
		return "", state, nil
	}
	return strings.TrimSpace(resp.Data.Relationships.AppStoreVersionForReview.Data.ID), state, nil
}

// reviewSubmissionStateProvesSubmission reports whether a fetched submission
// state proves the submission was actually sent to App Review. A draft
// (READY_FOR_REVIEW) or a withdrawal (CANCELING) contradicts a completed
// submit_review step, so the step must run again.
func reviewSubmissionStateProvesSubmission(state asc.ReviewSubmissionState) bool {
	switch state {
	case asc.ReviewSubmissionStateWaitingForReview,
		asc.ReviewSubmissionStateInReview,
		asc.ReviewSubmissionStateUnresolvedIssues,
		asc.ReviewSubmissionStateCompleting,
		asc.ReviewSubmissionStateComplete:
		return true
	default:
		return false
	}
}
