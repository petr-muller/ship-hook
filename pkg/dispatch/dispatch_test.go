package dispatch

import (
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"

	"github.com/petr-muller/boxship/pkg/config"
	"github.com/petr-muller/boxship/pkg/testhelpers"
)

type fakePlugin struct{}

func (f *fakePlugin) Name() string                                                       { return "fake" }
func (f *fakePlugin) HandlePullRequestEvent(_ *logrus.Entry, _ github.PullRequestEvent)   {}
func (f *fakePlugin) HandleIssueCommentEvent(_ *logrus.Entry, _ github.IssueCommentEvent) {}
func (f *fakePlugin) HandleReviewEvent(_ *logrus.Entry, _ github.ReviewEvent)             {}

type recordingPlugin struct {
	name               string
	mu                 sync.Mutex
	prEvents           []github.PullRequestEvent
	issueCommentEvents []github.IssueCommentEvent
	reviewEvents       []github.ReviewEvent
}

func (r *recordingPlugin) Name() string { return r.name }

func (r *recordingPlugin) HandlePullRequestEvent(_ *logrus.Entry, event github.PullRequestEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prEvents = append(r.prEvents, event)
}

func (r *recordingPlugin) HandleIssueCommentEvent(_ *logrus.Entry, event github.IssueCommentEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issueCommentEvents = append(r.issueCommentEvents, event)
}

func (r *recordingPlugin) HandleReviewEvent(_ *logrus.Entry, event github.ReviewEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reviewEvents = append(r.reviewEvents, event)
}

func enabledResolver(pluginNames ...string) *config.Resolver {
	var plugins []config.PluginConfig
	for _, name := range pluginNames {
		plugins = append(plugins, config.PluginConfig{Name: name})
	}
	return config.NewResolver(&config.Config{Plugins: plugins})
}

func TestDispatcherRegister(t *testing.T) {
	d := NewDispatcher(logrus.NewEntry(logrus.StandardLogger()), enabledResolver("fake"))
	d.Register(&fakePlugin{})

	if len(d.plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(d.plugins))
	}
	if d.plugins[0].Name() != "fake" {
		t.Errorf("expected plugin name 'fake', got %q", d.plugins[0].Name())
	}
}

func TestDispatcherHandlePullRequestEvent(t *testing.T) {
	d := NewDispatcher(logrus.NewEntry(logrus.StandardLogger()), enabledResolver("plugin-1", "plugin-2"))
	p1 := &recordingPlugin{name: "plugin-1"}
	p2 := &recordingPlugin{name: "plugin-2"}
	d.Register(p1)
	d.Register(p2)

	event := testhelpers.NewPullRequestEvent("org", "repo", 1, "opened")
	d.HandlePullRequestEvent(logrus.NewEntry(logrus.StandardLogger()), event)

	waitForEvents(t, func() int {
		p1.mu.Lock()
		p2.mu.Lock()
		defer p1.mu.Unlock()
		defer p2.mu.Unlock()
		if len(p1.prEvents) == 1 && len(p2.prEvents) == 1 {
			return 2
		}
		return 0
	}, 2)

	if p1.prEvents[0].Number != 1 {
		t.Errorf("plugin-1: expected PR #1, got #%d", p1.prEvents[0].Number)
	}
	if p2.prEvents[0].Number != 1 {
		t.Errorf("plugin-2: expected PR #1, got #%d", p2.prEvents[0].Number)
	}
}

func TestDispatcherHandleIssueCommentEvent(t *testing.T) {
	d := NewDispatcher(logrus.NewEntry(logrus.StandardLogger()), enabledResolver("recorder"))
	p := &recordingPlugin{name: "recorder"}
	d.Register(p)

	event := testhelpers.NewIssueCommentEvent("org", "repo", 5, "/test")
	d.HandleIssueCommentEvent(logrus.NewEntry(logrus.StandardLogger()), event)

	waitForEvents(t, func() int {
		p.mu.Lock()
		defer p.mu.Unlock()
		return len(p.issueCommentEvents)
	}, 1)

	if p.issueCommentEvents[0].Comment.Body != "/test" {
		t.Errorf("expected comment body '/test', got %q", p.issueCommentEvents[0].Comment.Body)
	}
}

func TestDispatcherDisabledPlugin(t *testing.T) {
	d := NewDispatcher(logrus.NewEntry(logrus.StandardLogger()), enabledResolver("enabled-plugin"))
	enabled := &recordingPlugin{name: "enabled-plugin"}
	disabled := &recordingPlugin{name: "disabled-plugin"}
	d.Register(enabled)
	d.Register(disabled)

	event := testhelpers.NewPullRequestEvent("org", "repo", 1, "opened")
	d.HandlePullRequestEvent(logrus.NewEntry(logrus.StandardLogger()), event)

	waitForEvents(t, func() int {
		enabled.mu.Lock()
		defer enabled.mu.Unlock()
		return len(enabled.prEvents)
	}, 1)

	// Give a short window for the disabled plugin to incorrectly receive events
	time.Sleep(50 * time.Millisecond)
	disabled.mu.Lock()
	defer disabled.mu.Unlock()
	if len(disabled.prEvents) != 0 {
		t.Errorf("disabled plugin should not have received events, got %d", len(disabled.prEvents))
	}
}

func TestDispatcherReviewEventGating(t *testing.T) {
	d := NewDispatcher(logrus.NewEntry(logrus.StandardLogger()), enabledResolver("enabled"))
	enabled := &recordingPlugin{name: "enabled"}
	disabled := &recordingPlugin{name: "disabled"}
	d.Register(enabled)
	d.Register(disabled)

	event := testhelpers.NewReviewEvent("org", "repo", 1, "submitted", "user", "APPROVED")
	d.HandleReviewEvent(logrus.NewEntry(logrus.StandardLogger()), event)

	waitForEvents(t, func() int {
		enabled.mu.Lock()
		defer enabled.mu.Unlock()
		return len(enabled.reviewEvents)
	}, 1)

	time.Sleep(50 * time.Millisecond)
	disabled.mu.Lock()
	defer disabled.mu.Unlock()
	if len(disabled.reviewEvents) != 0 {
		t.Errorf("disabled plugin should not have received review events, got %d", len(disabled.reviewEvents))
	}
}

func waitForEvents(t *testing.T, countFn func() int, expected int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if countFn() >= expected {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d events, got %d", expected, countFn())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
