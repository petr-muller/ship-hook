package config

import "fmt"

func (c *Config) MergeFrom(other *Config) error {
	if err := mergePluginLists(&c.Plugins, other.Plugins, "top-level"); err != nil {
		return err
	}

	if c.Orgs == nil && len(other.Orgs) > 0 {
		c.Orgs = make(map[string]OrgConfig)
	}

	for orgName, otherOrg := range other.Orgs {
		existing, exists := c.Orgs[orgName]
		if !exists {
			c.Orgs[orgName] = otherOrg
			continue
		}

		if err := mergePluginLists(&existing.Plugins, otherOrg.Plugins, fmt.Sprintf("org %q", orgName)); err != nil {
			return err
		}

		if existing.Repos == nil && len(otherOrg.Repos) > 0 {
			existing.Repos = make(map[string]RepoConfig)
		}

		for repoName, otherRepo := range otherOrg.Repos {
			existingRepo, repoExists := existing.Repos[repoName]
			if !repoExists {
				existing.Repos[repoName] = otherRepo
				continue
			}

			if err := mergePluginLists(&existingRepo.Plugins, otherRepo.Plugins, fmt.Sprintf("repo %q/%q", orgName, repoName)); err != nil {
				return err
			}
			existing.Repos[repoName] = existingRepo
		}

		c.Orgs[orgName] = existing
	}

	return nil
}

func mergePluginLists(dst *[]PluginConfig, src []PluginConfig, level string) error {
	for _, p := range src {
		if findPlugin(*dst, p.Name) != nil {
			return fmt.Errorf("duplicate plugin %q at %s", p.Name, level)
		}
		*dst = append(*dst, p)
	}
	return nil
}
