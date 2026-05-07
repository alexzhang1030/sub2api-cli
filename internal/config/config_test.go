package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSavesAndLoadsProfile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "config.toml"))

	profile := Profile{
		Name:           "work",
		BaseURL:        "https://sub2api.example.com/",
		Provider:       "github",
		Timezone:       "Asia/Shanghai",
		TokenExpiresAt: time.Unix(1893456000, 0).UTC(),
	}
	cfg := Config{CurrentProfile: "work", Profiles: map[string]Profile{"work": profile}}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	got, ok := loaded.Profiles["work"]
	if !ok {
		t.Fatalf("profile missing after load")
	}
	if loaded.CurrentProfile != "work" {
		t.Fatalf("current profile = %q", loaded.CurrentProfile)
	}
	if got.BaseURL != "https://sub2api.example.com" {
		t.Fatalf("base url = %q", got.BaseURL)
	}
	if got.Timezone != "Asia/Shanghai" {
		t.Fatalf("timezone = %q", got.Timezone)
	}
	if !got.TokenExpiresAt.Equal(profile.TokenExpiresAt) {
		t.Fatalf("expires at = %s", got.TokenExpiresAt)
	}
}

func TestTodayRangeUsesProfileTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	now := time.Date(2026, 5, 7, 13, 30, 0, 0, loc)

	start, end, label, err := TodayRange("Asia/Shanghai", now)
	if err != nil {
		t.Fatalf("today range: %v", err)
	}

	if label != "2026-05-07" {
		t.Fatalf("label = %q", label)
	}
	if start.Format(time.RFC3339) != "2026-05-07T00:00:00+08:00" {
		t.Fatalf("start = %s", start.Format(time.RFC3339))
	}
	if end.Format(time.RFC3339) != "2026-05-08T00:00:00+08:00" {
		t.Fatalf("end = %s", end.Format(time.RFC3339))
	}
}
