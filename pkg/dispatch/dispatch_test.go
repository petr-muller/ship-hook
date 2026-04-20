package dispatch

import (
	"testing"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"
)

type fakePlugin struct{}

func (f *fakePlugin) Name() string { return "fake" }
func (f *fakePlugin) HandlePullRequestEvent(_ *logrus.Entry, _ github.PullRequestEvent)  {}
func (f *fakePlugin) HandleIssueCommentEvent(_ *logrus.Entry, _ github.IssueCommentEvent) {}

func TestDispatcherRegister(t *testing.T) {
	d := NewDispatcher(logrus.NewEntry(logrus.StandardLogger()))
	d.Register(&fakePlugin{})

	if len(d.plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(d.plugins))
	}
	if d.plugins[0].Name() != "fake" {
		t.Errorf("expected plugin name 'fake', got %q", d.plugins[0].Name())
	}
}
