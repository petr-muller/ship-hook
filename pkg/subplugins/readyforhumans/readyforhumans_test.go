package readyforhumans

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"

	"github.com/openshift-eng/ship-hook/pkg/config"
)

type fakeClient struct {
	labelsAdded []string
}

func (c *fakeClient) AddLabel(org, repo string, number int, label string) error {
	c.labelsAdded = append(c.labelsAdded, fmt.Sprintf("%s/%s#%d:%s", org, repo, number, label))
	return nil
}

func nilResolver() *config.Resolver { return nil }

func resolverWithConfig(pluginName string, rawConfig string) *config.Resolver {
	return config.NewResolver(&config.Config{
		Plugins: []config.PluginConfig{{
			Name:   pluginName,
			Config: json.RawMessage(rawConfig),
		}},
	})
}

func TestHandleReviewEvent(t *testing.T) {
	testCases := []struct {
		name        string
		resolver    *config.Resolver
		event       github.ReviewEvent
		expectAdded []string
	}{
		{
			name:     "ignore review from different user",
			resolver: nilResolver(),
			event: github.ReviewEvent{
				Action: github.ReviewActionSubmitted,
				Review: github.Review{
					User:  github.User{Login: "someone-else"},
					State: github.ReviewStateApproved,
				},
				PullRequest: github.PullRequest{Number: 1},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
		},
		{
			name:     "ignore edited action from coderabbit",
			resolver: nilResolver(),
			event: github.ReviewEvent{
				Action: github.ReviewActionEdited,
				Review: github.Review{
					User:  github.User{Login: "coderabbitai[bot]"},
					State: github.ReviewStateApproved,
				},
				PullRequest: github.PullRequest{Number: 1},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
		},
		{
			name:     "ignore dismissed action from coderabbit",
			resolver: nilResolver(),
			event: github.ReviewEvent{
				Action: github.ReviewActionDismissed,
				Review: github.Review{
					User: github.User{Login: "coderabbitai[bot]"},
				},
				PullRequest: github.PullRequest{Number: 1},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
		},
		{
			name:     "ignore non-approved state",
			resolver: nilResolver(),
			event: github.ReviewEvent{
				Action: github.ReviewActionSubmitted,
				Review: github.Review{
					User:  github.User{Login: "coderabbitai[bot]"},
					State: github.ReviewStateChangesRequested,
				},
				PullRequest: github.PullRequest{Number: 42},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
		},
		{
			name:     "add label on approval when absent (default config)",
			resolver: nilResolver(),
			event: github.ReviewEvent{
				Action: github.ReviewActionSubmitted,
				Review: github.Review{
					User:  github.User{Login: "coderabbitai[bot]"},
					State: github.ReviewStateApproved,
				},
				PullRequest: github.PullRequest{Number: 42},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
			expectAdded: []string{"org/repo#42:ready-for-human-review"},
		},
		{
			name:     "no-op on approval when label already present",
			resolver: nilResolver(),
			event: github.ReviewEvent{
				Action: github.ReviewActionSubmitted,
				Review: github.Review{
					User:  github.User{Login: "coderabbitai[bot]"},
					State: github.ReviewStateApproved,
				},
				PullRequest: github.PullRequest{
					Number: 42,
					Labels: []github.Label{{Name: "ready-for-human-review"}},
				},
				Repo: github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
		},
		{
			name:     "custom label from config",
			resolver: resolverWithConfig(pluginName, `{"label":"custom-label"}`),
			event: github.ReviewEvent{
				Action: github.ReviewActionSubmitted,
				Review: github.Review{
					User:  github.User{Login: "coderabbitai[bot]"},
					State: github.ReviewStateApproved,
				},
				PullRequest: github.PullRequest{Number: 10},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
			expectAdded: []string{"org/repo#10:custom-label"},
		},
		{
			name:     "custom bot_login from config",
			resolver: resolverWithConfig(pluginName, `{"bot_login":"other-bot[bot]"}`),
			event: github.ReviewEvent{
				Action: github.ReviewActionSubmitted,
				Review: github.Review{
					User:  github.User{Login: "other-bot[bot]"},
					State: github.ReviewStateApproved,
				},
				PullRequest: github.PullRequest{Number: 10},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
			expectAdded: []string{"org/repo#10:ready-for-human-review"},
		},
		{
			name:     "custom bot_login does not match default bot",
			resolver: resolverWithConfig(pluginName, `{"bot_login":"other-bot[bot]"}`),
			event: github.ReviewEvent{
				Action: github.ReviewActionSubmitted,
				Review: github.Review{
					User:  github.User{Login: "coderabbitai[bot]"},
					State: github.ReviewStateApproved,
				},
				PullRequest: github.PullRequest{Number: 10},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakeClient{}
			p := New(fc, tc.resolver)
			p.HandleReviewEvent(context.Background(), logrus.NewEntry(logrus.StandardLogger()), tc.event)

			if len(tc.expectAdded) == 0 && len(fc.labelsAdded) == 0 {
				return
			}
			if fmt.Sprintf("%v", tc.expectAdded) != fmt.Sprintf("%v", fc.labelsAdded) {
				t.Errorf("labels added: want %v, got %v", tc.expectAdded, fc.labelsAdded)
			}
		})
	}
}
