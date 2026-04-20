package dispatch

import (
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"

	"github.com/petr-muller/boxship/pkg/config"
)

// SubPlugin defines the interface that all boxship sub-plugins must implement.
type SubPlugin interface {
	Name() string
	HandlePullRequestEvent(*logrus.Entry, github.PullRequestEvent)
	HandleIssueCommentEvent(*logrus.Entry, github.IssueCommentEvent)
	HandleReviewEvent(*logrus.Entry, github.ReviewEvent)
}

type Dispatcher struct {
	plugins  []SubPlugin
	resolver *config.Resolver
	logger   *logrus.Entry
}

func NewDispatcher(logger *logrus.Entry, resolver *config.Resolver) *Dispatcher {
	return &Dispatcher{
		resolver: resolver,
		logger:   logger,
	}
}

func (d *Dispatcher) Register(p SubPlugin) {
	d.logger.WithField("plugin", p.Name()).Info("Registering sub-plugin")
	d.plugins = append(d.plugins, p)
}

func (d *Dispatcher) HandlePullRequestEvent(l *logrus.Entry, event github.PullRequestEvent) {
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	for _, p := range d.plugins {
		plugin := p
		if !d.resolver.IsEnabled(plugin.Name(), org, repo) {
			continue
		}
		go plugin.HandlePullRequestEvent(l.WithField("plugin", plugin.Name()), event)
	}
}

func (d *Dispatcher) HandleIssueCommentEvent(l *logrus.Entry, event github.IssueCommentEvent) {
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	for _, p := range d.plugins {
		plugin := p
		if !d.resolver.IsEnabled(plugin.Name(), org, repo) {
			continue
		}
		go plugin.HandleIssueCommentEvent(l.WithField("plugin", plugin.Name()), event)
	}
}

func (d *Dispatcher) HandleReviewEvent(l *logrus.Entry, event github.ReviewEvent) {
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	for _, p := range d.plugins {
		plugin := p
		if !d.resolver.IsEnabled(plugin.Name(), org, repo) {
			continue
		}
		go plugin.HandleReviewEvent(l.WithField("plugin", plugin.Name()), event)
	}
}
