// Package example is a reference sub-plugin implementation.
// Copy this package to create new sub-plugins.
package example

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"
)

type githubClient interface {
	CreateComment(owner, repo string, number int, comment string) error
}

type Plugin struct {
	ghc githubClient
}

func New(ghc githubClient) *Plugin {
	return &Plugin{ghc: ghc}
}

func (p *Plugin) Name() string {
	return "example"
}

func (p *Plugin) HandlePullRequestEvent(_ context.Context, l *logrus.Entry, event github.PullRequestEvent) {
	l.Info("Received pull request event")
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	number := event.Number
	if err := p.ghc.CreateComment(org, repo, number, fmt.Sprintf("example plugin noticed PR #%d", number)); err != nil {
		l.WithError(err).Error("Failed to create comment")
	}
}

func (p *Plugin) HandleIssueCommentEvent(_ context.Context, l *logrus.Entry, event github.IssueCommentEvent) {
	l.Info("Received issue comment event")
}

func (p *Plugin) HandleReviewEvent(_ context.Context, l *logrus.Entry, event github.ReviewEvent) {
	l.Info("Received review event")
}
