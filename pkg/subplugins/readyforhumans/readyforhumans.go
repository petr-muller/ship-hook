package readyforhumans

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"

	"github.com/openshift-eng/ship-hook/pkg/config"
	"github.com/openshift-eng/ship-hook/pkg/dispatch"
)

const pluginName = "ready-for-humans"

type PluginConfig struct {
	Label    string `json:"label"`
	BotLogin string `json:"bot_login"`
}

func defaultConfig() PluginConfig {
	return PluginConfig{
		Label:    "ready-for-human-review",
		BotLogin: "coderabbitai[bot]",
	}
}

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
}

type Plugin struct {
	ghc      githubClient
	resolver *config.Resolver
}

func New(ghc githubClient, resolver *config.Resolver) *Plugin {
	return &Plugin{ghc: ghc, resolver: resolver}
}

func (p *Plugin) Name() string { return pluginName }

func (p *Plugin) HandlePullRequestEvent(_ context.Context, _ *logrus.Entry, _ github.PullRequestEvent) dispatch.HandlerResult {
	return dispatch.Irrelevant("only handles review events")
}

func (p *Plugin) HandleIssueCommentEvent(_ context.Context, _ *logrus.Entry, _ github.IssueCommentEvent) dispatch.HandlerResult {
	return dispatch.Irrelevant("only handles review events")
}

func (p *Plugin) HandleReviewEvent(_ context.Context, l *logrus.Entry, re github.ReviewEvent) dispatch.HandlerResult {
	org := re.Repo.Owner.Login
	repo := re.Repo.Name
	cfg := config.ResolvePluginConfig(p.resolver, pluginName, defaultConfig(), org, repo)

	if re.Review.User.Login != cfg.BotLogin {
		return dispatch.Irrelevant("reviewer is not the configured bot")
	}

	if re.Action != github.ReviewActionSubmitted {
		return dispatch.Irrelevant("action is not submitted")
	}

	if !strings.EqualFold(string(re.Review.State), string(github.ReviewStateApproved)) {
		return dispatch.Irrelevant("review state is not approved")
	}

	if prHasLabel(re.PullRequest, cfg.Label) {
		return dispatch.Irrelevant("PR already has label")
	}

	number := re.PullRequest.Number

	if err := p.ghc.AddLabel(org, repo, number, cfg.Label); err != nil {
		l.WithError(err).Error("Failed to add ready-for-humans label")
	}
	return dispatch.Handled("added ready-for-humans label")
}

func prHasLabel(pr github.PullRequest, label string) bool {
	for _, l := range pr.Labels {
		if strings.EqualFold(l.Name, label) {
			return true
		}
	}
	return false
}
