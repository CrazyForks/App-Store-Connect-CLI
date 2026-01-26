package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	configDirName    = ".asc"
	configFileName   = "config.json"
	configPathEnvVar = "ASC_CONFIG_PATH"
)

// DurationValue stores a duration with its raw string representation.
// It marshals to/from JSON as a string to preserve config compatibility.
type DurationValue struct {
	Duration time.Duration
	Raw      string
}

// ParseDurationValue parses a duration string or seconds value into a DurationValue.
func ParseDurationValue(raw string) (DurationValue, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DurationValue{}, nil
	}
	parsed, err := parseDurationValue(raw)
	if err != nil {
		return DurationValue{}, err
	}
	return DurationValue{Duration: parsed, Raw: raw}, nil
}

// Value returns the parsed duration if it's positive.
func (d DurationValue) Value() (time.Duration, bool) {
	if d.Duration > 0 {
		return d.Duration, true
	}
	raw := strings.TrimSpace(d.Raw)
	if raw == "" {
		return 0, false
	}
	parsed, err := parseDurationValue(raw)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

// String returns the raw string when available, falling back to the duration value.
func (d DurationValue) String() string {
	if strings.TrimSpace(d.Raw) != "" {
		return d.Raw
	}
	if d.Duration == 0 {
		return ""
	}
	return d.Duration.String()
}

// MarshalJSON stores the raw string when available, preserving the config format.
func (d DurationValue) MarshalJSON() ([]byte, error) {
	raw := strings.TrimSpace(d.Raw)
	if raw == "" {
		if d.Duration == 0 {
			return json.Marshal("")
		}
		raw = d.Duration.String()
	}
	return json.Marshal(raw)
}

// UnmarshalJSON parses duration strings or seconds values from JSON.
func (d *DurationValue) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	d.Raw = raw
	if raw == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := parseDurationValue(raw)
	if err != nil {
		d.Duration = 0
		return nil
	}
	d.Duration = parsed
	return nil
}

func parseDurationValue(raw string) (time.Duration, error) {
	if parsed, err := time.ParseDuration(raw); err == nil {
		return parsed, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}
	return time.Duration(seconds) * time.Second, nil
}

// Credential stores a named API credential in config.json.
type Credential struct {
	Name           string `json:"name"`
	KeyID          string `json:"key_id"`
	IssuerID       string `json:"issuer_id"`
	PrivateKeyPath string `json:"private_key_path"`
}

// Config holds the application configuration
type Config struct {
	KeyID          string       `json:"key_id"`
	IssuerID       string       `json:"issuer_id"`
	PrivateKeyPath string       `json:"private_key_path"`
	DefaultKeyName string       `json:"default_key_name"`
	Keys           []Credential `json:"keys,omitempty"`
	AppID          string       `json:"app_id"`

	VendorNumber          string `json:"vendor_number"`
	AnalyticsVendorNumber string `json:"analytics_vendor_number"`

	Timeout              DurationValue `json:"timeout"`
	TimeoutSeconds       DurationValue `json:"timeout_seconds"`
	UploadTimeout        DurationValue `json:"upload_timeout"`
	UploadTimeoutSeconds DurationValue `json:"upload_timeout_seconds"`
	MaxRetries           string        `json:"max_retries"`
	BaseDelay            string        `json:"base_delay"`
	MaxDelay             string        `json:"max_delay"`
	RetryLog             string        `json:"retry_log"`
}

// ErrNotFound is returned when the config file doesn't exist
var ErrNotFound = fmt.Errorf("configuration not found")

// configDir returns the path to the configuration directory
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configDirName), nil
}

// configPath returns the path to the config file
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// GlobalPath returns the global configuration file path.
func GlobalPath() (string, error) {
	return configPath()
}

// Path returns the active configuration file path.
func Path() (string, error) {
	return resolvePath()
}

// LocalPath returns the local configuration file path.
func LocalPath() (string, error) {
	baseDir, err := localConfigBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, configDirName, configFileName), nil
}

func resolvePath() (string, error) {
	if envPath := strings.TrimSpace(os.Getenv(configPathEnvVar)); envPath != "" {
		return filepath.Clean(envPath), nil
	}

	localPath, err := findLocalConfigPath()
	if err != nil {
		return "", err
	}
	if localPath != "" {
		return localPath, nil
	}

	return GlobalPath()
}

func findLocalConfigPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, configDirName, configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat config: %w", err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func localConfigBaseDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	dir := cwd
	for {
		ascDir := filepath.Join(dir, configDirName)
		if info, err := os.Stat(ascDir); err == nil {
			if info.IsDir() {
				return dir, nil
			}
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat %s: %w", ascDir, err)
		}

		gitEntry := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitEntry); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat %s: %w", gitEntry, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd, nil
		}
		dir = parent
	}
}

// Load loads the configuration from the config file
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadAt(path)
}

// Save saves the configuration to the config file
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return SaveAt(path, cfg)
}

// Remove removes the config file
func Remove() error {
	path, err := Path()
	if err != nil {
		return err
	}
	return RemoveAt(path)
}

// LoadAt loads the configuration from the provided path.
func LoadAt(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("failed to read config: empty path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// SaveAt saves the configuration to the provided path.
func SaveAt(path string, cfg *Config) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("failed to write config: empty path")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// RemoveAt removes the config file at the provided path.
func RemoveAt(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("failed to remove config: empty path")
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove config: %w", err)
	}

	return nil
}
