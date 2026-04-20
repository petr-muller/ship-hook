// Package example is a reference sub-plugin implementation.
// Copy this package to create new sub-plugins.
package example

import (
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "example"
}

func (p *Plugin) HandlePullRequestEvent(l *logrus.Entry, event github.PullRequestEvent) {
	l.Info("Received pull request event")
}

func (p *Plugin) HandleIssueCommentEvent(l *logrus.Entry, event github.IssueCommentEvent) {
	l.Info("Received issue comment event")
}
