package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	DefaultProfile = "default"
	DefaultAppDir  = "sub2api-cli"
)

type Config struct {
	CurrentProfile string             `toml:"current_profile"`
	Profiles       map[string]Profile `toml:"profiles"`
}

type Profile struct {
	Name           string    `toml:"name"`
	BaseURL        string    `toml:"base_url"`
	Provider       string    `toml:"provider"`
	Timezone       string    `toml:"timezone"`
	TokenExpiresAt time.Time `toml:"token_expires_at"`
}

type Store struct {
	path string
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultAppDir, "config.toml"), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (Config, error) {
	cfg := Config{CurrentProfile: DefaultProfile, Profiles: map[string]Profile{}}
	if strings.TrimSpace(s.path) == "" {
		return cfg, errors.New("config path is empty")
	}
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(s.path, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if strings.TrimSpace(cfg.CurrentProfile) == "" {
		cfg.CurrentProfile = DefaultProfile
	}
	for name, profile := range cfg.Profiles {
		profile.Name = name
		profile.BaseURL = NormalizeBaseURL(profile.BaseURL)
		if strings.TrimSpace(profile.Timezone) == "" {
			profile.Timezone = LocalTimezone()
		}
		cfg.Profiles[name] = profile
	}
	return cfg, nil
}

func (s *Store) Save(cfg Config) error {
	if strings.TrimSpace(s.path) == "" {
		return errors.New("config path is empty")
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if strings.TrimSpace(cfg.CurrentProfile) == "" {
		cfg.CurrentProfile = DefaultProfile
	}
	for name, profile := range cfg.Profiles {
		profile.Name = name
		profile.BaseURL = NormalizeBaseURL(profile.BaseURL)
		if strings.TrimSpace(profile.Timezone) == "" {
			profile.Timezone = LocalTimezone()
		}
		cfg.Profiles[name] = profile
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func (c Config) Current() (Profile, error) {
	name := strings.TrimSpace(c.CurrentProfile)
	if name == "" {
		name = DefaultProfile
	}
	profile, ok := c.Profiles[name]
	if !ok {
		return Profile{}, errors.New("profile not found; run sub2api login")
	}
	profile.Name = name
	return profile, nil
}

func NormalizeBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func LocalTimezone() string {
	if name := time.Now().Location().String(); name != "" && name != "Local" {
		return name
	}
	return "UTC"
}

func TodayRange(timezone string, now time.Time) (time.Time, time.Time, string, error) {
	if strings.TrimSpace(timezone) == "" {
		timezone = LocalTimezone()
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)
	return start, end, start.Format("2006-01-02"), nil
}
