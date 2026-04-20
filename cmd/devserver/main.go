package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
	"sigs.k8s.io/prow/pkg/githubeventserver"
	"sigs.k8s.io/prow/pkg/logrusutil"

	"github.com/petr-muller/boxship/pkg/config"
	"github.com/petr-muller/boxship/pkg/dispatch"
	"github.com/petr-muller/boxship/pkg/subplugins/example"
	"github.com/petr-muller/boxship/pkg/subplugins/readyforhumans"
)

const devHMAC = "devhmac"

type stateResponse struct {
	IssueCommentsAdded []string `json:"issue_comments_added"`
	IssueLabelsAdded   []string `json:"issue_labels_added"`
	IssueLabelsRemoved []string `json:"issue_labels_removed"`
	AssigneesAdded     []string `json:"assignees_added"`
}

func main() {
	logrusutil.ComponentInit()
	logger := logrus.WithField("component", "boxship-devserver")

	var statePort int
	var configPath string
	esOpts := githubeventserver.Options{}

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&statePort, "state-port", 8889, "State API port")
	fs.StringVar(&configPath, "config-path", "", "Path to boxship config file (default: all plugins enabled)")
	esOpts.Bind(fs)
	if err := fs.Parse(os.Args[1:]); err != nil {
		logger.WithError(err).Fatal("Failed to parse flags")
	}

	if err := esOpts.DefaultAndValidate(); err != nil {
		logger.WithError(err).Fatal("Invalid event server options")
	}

	var resolver *config.Resolver
	if configPath != "" {
		cfg, err := config.Load(configPath, "")
		if err != nil {
			logger.WithError(err).Fatal("Failed to load config")
		}
		resolver = config.NewResolver(cfg)
	} else {
		resolver = config.NewResolver(&config.Config{
			Plugins: []config.PluginConfig{
				{Name: "example"},
				{Name: "ready-for-humans"},
			},
		})
	}

	ghc := fakegithub.NewFakeClient()

	hmacTokenGenerator := func() []byte { return []byte(devHMAC) }
	eventServer := githubeventserver.New(esOpts, hmacTokenGenerator, logger)

	dispatcher := dispatch.NewDispatcher(logger, resolver)
	dispatcher.Register(example.New(ghc))
	dispatcher.Register(readyforhumans.New(ghc, resolver))

	eventServer.RegisterHandlePullRequestEvent(dispatcher.HandlePullRequestEvent)
	eventServer.RegisterHandleIssueCommentEvent(dispatcher.HandleIssueCommentEvent)
	eventServer.RegisterReviewEventHandler(dispatcher.HandleReviewEvent)

	stateMux := http.NewServeMux()
	stateMux.HandleFunc("GET /state", func(w http.ResponseWriter, r *http.Request) {
		state := stateResponse{
			IssueCommentsAdded: ghc.IssueCommentsAdded,
			IssueLabelsAdded:   ghc.IssueLabelsAdded,
			IssueLabelsRemoved: ghc.IssueLabelsRemoved,
			AssigneesAdded:     ghc.AssigneesAdded,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)
	})
	stateMux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		ghc.IssueCommentsAdded = nil
		ghc.IssueLabelsAdded = nil
		ghc.IssueLabelsRemoved = nil
		ghc.AssigneesAdded = nil
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "state reset")
	})

	go func() {
		logger.Infof("State API listening on :%d", statePort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", statePort), stateMux); err != nil {
			logger.WithError(err).Fatal("State API server failed")
		}
	}()

	logger.Info("Dev webhook server starting")
	if err := eventServer.ListenAndServe(); err != nil {
		logger.WithError(err).Fatal("Event server failed")
	}
}
