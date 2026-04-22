package dispatch

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"

	"github.com/petr-muller/boxship/pkg/config"
)

// SubPlugin defines the interface that all boxship sub-plugins must implement.
type SubPlugin interface {
	Name() string
	HandlePullRequestEvent(context.Context, *logrus.Entry, github.PullRequestEvent)
	HandleIssueCommentEvent(context.Context, *logrus.Entry, github.IssueCommentEvent)
	HandleReviewEvent(context.Context, *logrus.Entry, github.ReviewEvent)
}

// Dispatcher multiplexes GitHub webhook events to registered sub-plugins.
// It holds a cancellable context to bridge the gap between Prow's context-free
// handler signatures and sub-plugins that need shutdown signaling.
// See specs/005-graceful-shutdown.md for the rationale.
type Dispatcher struct {
	plugins  []SubPlugin
	resolver *config.Resolver
	logger   *logrus.Entry
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewDispatcher(logger *logrus.Entry, resolver *config.Resolver) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		resolver: resolver,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
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
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			plugin.HandlePullRequestEvent(d.ctx, l.WithField("plugin", plugin.Name()), event)
		}()
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
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			plugin.HandleIssueCommentEvent(d.ctx, l.WithField("plugin", plugin.Name()), event)
		}()
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
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			plugin.HandleReviewEvent(d.ctx, l.WithField("plugin", plugin.Name()), event)
		}()
	}
}

// Shutdown signals all in-flight handlers to stop and waits for them to finish.
// Returns the context error if the provided context expires before all handlers complete.
func (d *Dispatcher) Shutdown(ctx context.Context) error {
	d.cancel()
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
