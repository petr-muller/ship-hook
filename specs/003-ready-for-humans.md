# Ready for Humans

## Status
Implemented

## Motivation

CodeRabbit (`coderabbitai[bot]`) performs automated code reviews on pull requests. Human reviewers benefit from knowing when CodeRabbit has finished its review and approved the PR, so they can focus their time on PRs that are ready for human attention. A `ready-for-humans` label on the PR signals this.

## Design

### Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `label` | `ready-for-human-review` | Label to add to the PR |
| `bot_login` | `coderabbitai[bot]` | GitHub login of the review bot to watch |

Example config:
```yaml
plugins:
  - name: ready-for-humans
    config:
      label: ai-reviewed
      bot_login: "coderabbitai[bot]"
```

### Trigger

The plugin handles `ReviewEvent` from GitHub (webhook event type `pull_request_review`). It acts when all of the following are true:

- The review author matches `bot_login`
- The action is `submitted`
- The review state is `APPROVED`
- The PR does not already have the configured label

When triggered, the plugin adds the configured label to the PR.

### Scope

This is intentionally minimal. The plugin only adds the label on approval. It does not:

- Remove the label when CodeRabbit submits a non-approving review
- Remove the label when new commits are pushed
- React to dismissed reviews

### Infrastructure Extension

The `dispatch.SubPlugin` interface was extended with `HandleReviewEvent(*logrus.Entry, github.ReviewEvent)` to support this plugin. The Dispatcher and event server registration were updated accordingly.

### Package Layout

- `pkg/subplugins/readyforhumans/readyforhumans.go` — plugin implementation
- `pkg/subplugins/readyforhumans/readyforhumans_test.go` — unit tests
- `test/integration/plugins/readyforhumans/readyforhumans_test.go` — integration test
- `test/integration/testdata/review_submitted_approved.json` — test payload

### GitHub Client Interface

The plugin defines a narrow interface requiring only `AddLabel(org, repo string, number int, label string) error`.

## Verification

1. `make test` — unit tests cover: ignoring non-CodeRabbit users, non-submitted actions, non-approved states, idempotent label addition
2. `make integration-test` — integration test loads a realistic webhook payload and verifies the label is added
3. `make dev-webhook EVENT=pull_request_review PAYLOAD=test/integration/testdata/review_submitted_approved.json` followed by `make dev-state` shows the label in `issue_labels_added`
