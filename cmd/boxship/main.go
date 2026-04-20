package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/flagutil"
	"sigs.k8s.io/prow/pkg/githubeventserver"
	"sigs.k8s.io/prow/pkg/interrupts"
	"sigs.k8s.io/prow/pkg/logrusutil"

	"github.com/petr-muller/boxship/pkg/dispatch"
	"github.com/petr-muller/boxship/pkg/subplugins/example"
)

type options struct {
	eventServerOptions githubeventserver.Options
	github             flagutil.GitHubOptions
	hmacSecretFile     string
	dryRun             bool
	logLevel           string
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	o.eventServerOptions.Bind(fs)
	o.github.AddFlags(fs)
	fs.StringVar(&o.hmacSecretFile, "hmac-secret-file", "/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing (uses no mutations)")
	fs.StringVar(&o.logLevel, "log-level", "debug", "Log level (trace, debug, info, warn, error, fatal, panic)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
		os.Exit(1)
	}

	return o
}

func (o *options) validate() error {
	level, err := logrus.ParseLevel(o.logLevel)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", o.logLevel, err)
	}
	logrus.SetLevel(level)

	if err := o.eventServerOptions.DefaultAndValidate(); err != nil {
		return fmt.Errorf("invalid event server options: %w", err)
	}

	return nil
}

func main() {
	logrusutil.ComponentInit()
	logger := logrus.WithField("component", "boxship")

	o := gatherOptions()
	if err := o.validate(); err != nil {
		logger.WithError(err).Fatal("Invalid options")
	}

	hmacTokenGenerator := func() []byte {
		data, err := os.ReadFile(o.hmacSecretFile)
		if err != nil {
			logger.WithError(err).Error("Failed to read HMAC secret file")
			return nil
		}
		return data
	}

	eventServer := githubeventserver.New(o.eventServerOptions, hmacTokenGenerator, logger)

	dispatcher := dispatch.NewDispatcher(logger)
	dispatcher.Register(example.New())

	eventServer.RegisterHandlePullRequestEvent(dispatcher.HandlePullRequestEvent)
	eventServer.RegisterHandleIssueCommentEvent(dispatcher.HandleIssueCommentEvent)

	interrupts.OnInterrupt(func() {
		eventServer.GracefulShutdown()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := eventServer.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("Error shutting down event server")
		}
	})

	if err := eventServer.ListenAndServe(); err != nil {
		logger.WithError(err).Fatal("Event server failed")
	}
}
