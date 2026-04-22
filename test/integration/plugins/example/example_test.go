//go:build integration

package example

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"

	"github.com/petr-muller/boxship/pkg/subplugins/example"
	integration "github.com/petr-muller/boxship/test/integration"
)

func TestExamplePlugin_HandlePullRequestEvent(t *testing.T) {
	ghc := fakegithub.NewFakeClient()
	ghc.OrgMembers = map[string][]string{"test-org": {"test-author"}}
	ghc.RepoLabelsExisting = []string{"needs-review", "approved", "lgtm"}

	plugin := example.New(ghc)
	event := integration.LoadTestEvent[github.PullRequestEvent](t, "pull_request_opened.json")

	plugin.HandlePullRequestEvent(context.Background(), logrus.NewEntry(logrus.StandardLogger()), event)

	if len(ghc.IssueCommentsAdded) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(ghc.IssueCommentsAdded))
	}
	expected := "test-org/test-repo#1:example plugin noticed PR #1"
	if ghc.IssueCommentsAdded[0] != expected {
		t.Errorf("expected comment %q, got %q", expected, ghc.IssueCommentsAdded[0])
	}
}

func TestExamplePlugin_HandleIssueCommentEvent(t *testing.T) {
	ghc := fakegithub.NewFakeClient()
	plugin := example.New(ghc)

	event := integration.LoadTestEvent[github.IssueCommentEvent](t, "issue_comment_created.json")

	plugin.HandleIssueCommentEvent(context.Background(), logrus.NewEntry(logrus.StandardLogger()), event)

	if len(ghc.IssueCommentsAdded) != 0 {
		t.Errorf("expected no comments from issue comment handler, got %d", len(ghc.IssueCommentsAdded))
	}
}
