# Configuration

## Status
Implemented

## Motivation

Boxship needs a configuration system so that sub-plugins can be enabled or disabled per org/repo, and sub-plugins can receive custom configuration that varies across organizations and repositories. The system must support supplemental config files so that org/repo owners can shard their configuration into separate YAML files.

## Design

### Config File Format

The config is YAML with three levels of hierarchy: top-level (global defaults), org-level, and repo-level. Lower levels override upper levels.

```yaml
plugins:
  - name: ready-for-humans
    config:
      label: ready-for-humans
      bot_login: "coderabbitai[bot]"
  - name: example

orgs:
  openshift:
    plugins:
      - name: ready-for-humans
        config:
          label: openshift-ready
    repos:
      ci-tools:
        plugins:
          - name: ready-for-humans
            disabled: true
```

### Enable/Disable Semantics

- Listing a plugin means it is enabled
- Use `disabled: true` to explicitly disable a plugin at a given level
- If a plugin is not listed at any level, it is disabled
- Lower levels override upper: repo > org > top-level

### Plugin Config Merging

Plugin config is merged per-field across layers. Each layer only overrides the fields it specifies; unspecified fields inherit from the layer above or from the plugin's built-in defaults.

Plugins receive their resolved, typed config via `config.ResolvePluginConfig[T]()`. This generic function starts with the plugin's default config struct, then applies each layer's raw config via `json.Unmarshal`, which naturally gives per-field override semantics.

### Supplemental Config Files

Additional YAML files in a supplemental directory are merged into the main config via `Config.MergeFrom()`. Supplemental configs have the same structure as the main config. Duplicate plugin entries at the same level (top-level, same org, same repo) across files are rejected as errors.

### Key Types

- `pkg/config/config.go` — `Config`, `PluginConfig`, `OrgConfig`, `RepoConfig`, `Load()`
- `pkg/config/merge.go` — `Config.MergeFrom()`
- `pkg/config/resolve.go` — `Resolver`, `ResolvePluginConfig[T]()`

### CLI Flags

- `--config-path` — path to the main config file
- `--supplemental-config-dir` — path to a directory of supplemental YAML config files

### Runtime Integration

The `Dispatcher` receives a `*config.Resolver` and checks `IsEnabled()` before dispatching events to each plugin. Plugins that need config call `config.ResolvePluginConfig()` in their handlers to get a typed config struct for the event's org/repo.

## Verification

1. `make verify` passes
2. `pkg/config/config_test.go` covers loading, merging, enabled resolution, and typed config resolution
3. Dev server: `make dev-server` with `--config-path` applies custom config; without it, all plugins are enabled by default
