package readyforhumans

import (
	"strings"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"

	"github.com/petr-muller/boxship/pkg/config"
)

const pluginName = "ready-for-humans"

type PluginConfig struct {
	Label    string `json:"label"`
	BotLogin string `json:"bot_login"`
}

func defaultConfig() PluginConfig {
	return PluginConfig{
		Label:    "ready-for-humans",
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

func (p *Plugin) HandlePullRequestEvent(_ *logrus.Entry, _ github.PullRequestEvent) {}

func (p *Plugin) HandleIssueCommentEvent(_ *logrus.Entry, _ github.IssueCommentEvent) {}

func (p *Plugin) HandleReviewEvent(l *logrus.Entry, re github.ReviewEvent) {
	org := re.Repo.Owner.Login
	repo := re.Repo.Name
	cfg := config.ResolvePluginConfig(p.resolver, pluginName, defaultConfig(), org, repo)

	if re.Review.User.Login != cfg.BotLogin {
		return
	}

	if re.Action != github.ReviewActionSubmitted {
		return
	}

	if re.Review.State != github.ReviewStateApproved {
		return
	}

	if prHasLabel(re.PullRequest, cfg.Label) {
		return
	}

	number := re.PullRequest.Number

	if err := p.ghc.AddLabel(org, repo, number, cfg.Label); err != nil {
		l.WithError(err).Error("Failed to add ready-for-humans label")
	}
}

func prHasLabel(pr github.PullRequest, label string) bool {
	for _, l := range pr.Labels {
		if strings.EqualFold(l.Name, label) {
			return true
		}
	}
	return false
}
