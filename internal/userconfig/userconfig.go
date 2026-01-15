package userconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigDir is the directory under $HOME for wt config
	ConfigDir = ".config/wt"
	// ConfigFile is the name of the user config file
	ConfigFile = "config.yaml"
)

// RepoConfig holds per-repository user settings
type RepoConfig struct {
	Remote        string  `yaml:"remote,omitempty"`
	FetchInterval *string `yaml:"fetch_interval,omitempty"` // pointer to distinguish unset from empty
}

// UserConfig holds user-level configuration
type UserConfig struct {
	// Remote is the default remote to compare against (empty = local comparison)
	Remote string `yaml:"remote,omitempty"`
	// FetchInterval is the minimum time between fetches (e.g., "5m", "1h", "never")
	FetchInterval string `yaml:"fetch_interval,omitempty"`
	// Repos holds per-repository overrides keyed by absolute repo path
	Repos map[string]RepoConfig `yaml:"repos,omitempty"`
}

// DefaultFetchInterval is the default minimum time between fetches
const DefaultFetchInterval = "5m"

// DefaultUserConfig returns a config with default values
func DefaultUserConfig() *UserConfig {
	return &UserConfig{
		Remote:        "",                   // default to local comparison
		FetchInterval: DefaultFetchInterval, // default 5 minutes between fetches
		Repos:         make(map[string]RepoConfig),
	}
}

// GetConfigPath returns the full path to the user config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDir, ConfigFile), nil
}

// Load reads user config from ~/.config/wt/config.yaml
// Returns default config if file doesn't exist
func Load() (*UserConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return DefaultUserConfig(), err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultUserConfig(), nil
		}
		return DefaultUserConfig(), err
	}

	cfg := DefaultUserConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return DefaultUserConfig(), fmt.Errorf("failed to parse config: %w", err)
	}

	// Ensure Repos map is initialized
	if cfg.Repos == nil {
		cfg.Repos = make(map[string]RepoConfig)
	}

	return cfg, nil
}

// Save writes user config to ~/.config/wt/config.yaml
// Uses atomic write (temp file + rename) to prevent corruption if interrupted.
func Save(cfg *UserConfig) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temp file first for atomic save
	tempFile, err := os.CreateTemp(dir, ".config.yaml.tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	success = true
	return nil
}

// GetRemoteForRepo returns the effective remote for a given repo path
// Returns per-repo override if set, otherwise global default
func (c *UserConfig) GetRemoteForRepo(repoPath string) string {
	if repoConfig, ok := c.Repos[repoPath]; ok && repoConfig.Remote != "" {
		return repoConfig.Remote
	}
	return c.Remote
}

// FetchIntervalNever is a sentinel value indicating fetch is disabled
const FetchIntervalNever = time.Duration(-1)

// GetFetchIntervalForRepo returns the effective fetch interval for a given repo path
// Returns per-repo override if set, otherwise global default, otherwise DefaultFetchInterval
// Returns FetchIntervalNever (-1) if set to "never"
func (c *UserConfig) GetFetchIntervalForRepo(repoPath string) time.Duration {
	intervalStr := c.FetchInterval
	if intervalStr == "" {
		intervalStr = DefaultFetchInterval
	}

	// Check for per-repo override
	if repoConfig, ok := c.Repos[repoPath]; ok && repoConfig.FetchInterval != nil {
		intervalStr = *repoConfig.FetchInterval
	}

	// Handle "never" as a special case
	if intervalStr == "never" {
		return FetchIntervalNever
	}

	// Parse duration, return 0 on error (which means always fetch)
	d, _ := time.ParseDuration(intervalStr)
	return d
}

// SetGlobal sets a global config value
func (c *UserConfig) SetGlobal(key, value string) error {
	switch key {
	case "remote":
		c.Remote = value
	case "fetch_interval":
		c.FetchInterval = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// UnsetGlobal clears a global config value to its default
func (c *UserConfig) UnsetGlobal(key string) error {
	switch key {
	case "remote":
		c.Remote = ""
	case "fetch_interval":
		c.FetchInterval = ""
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// SetForRepo sets a per-repo config value
func (c *UserConfig) SetForRepo(repoPath, key, value string) error {
	if c.Repos == nil {
		c.Repos = make(map[string]RepoConfig)
	}

	repoConfig := c.Repos[repoPath]

	switch key {
	case "remote":
		repoConfig.Remote = value
	case "fetch_interval":
		repoConfig.FetchInterval = &value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	c.Repos[repoPath] = repoConfig
	return nil
}

// UnsetForRepo removes a per-repo config value
func (c *UserConfig) UnsetForRepo(repoPath, key string) error {
	if c.Repos == nil {
		return nil
	}

	repoConfig, ok := c.Repos[repoPath]
	if !ok {
		return nil
	}

	switch key {
	case "remote":
		repoConfig.Remote = ""
	case "fetch_interval":
		repoConfig.FetchInterval = nil
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	// If repo config is now empty, remove it entirely
	if repoConfig.Remote == "" && repoConfig.FetchInterval == nil {
		delete(c.Repos, repoPath)
	} else {
		c.Repos[repoPath] = repoConfig
	}

	return nil
}

// GetGlobal returns a global config value as a string
func (c *UserConfig) GetGlobal(key string) (string, error) {
	switch key {
	case "remote":
		return c.Remote, nil
	case "fetch_interval":
		if c.FetchInterval != "" {
			return c.FetchInterval, nil
		}
		return DefaultFetchInterval, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// GetForRepo returns a per-repo config value as a string
// Returns empty string and false if not set
func (c *UserConfig) GetForRepo(repoPath, key string) (string, bool) {
	repoConfig, ok := c.Repos[repoPath]
	if !ok {
		return "", false
	}

	switch key {
	case "remote":
		if repoConfig.Remote != "" {
			return repoConfig.Remote, true
		}
	case "fetch_interval":
		if repoConfig.FetchInterval != nil {
			return *repoConfig.FetchInterval, true
		}
	}

	return "", false
}

// ValidKeys returns the list of valid configuration keys
func ValidKeys() []string {
	return []string{"remote", "fetch_interval"}
}
