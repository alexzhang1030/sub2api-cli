package auth

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type BrowserTokenExport struct {
	AccessToken  string
	RefreshToken string
	ExpiresAtMS  int64
}

func ParseBrowserTokenExport(raw []byte) (BrowserTokenExport, error) {
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return BrowserTokenExport{}, err
	}
	out := BrowserTokenExport{
		AccessToken:  firstString(values, "auth_token", "access_token"),
		RefreshToken: firstString(values, "refresh_token"),
		ExpiresAtMS:  firstInt64(values, "token_expires_at", "expires_at"),
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return BrowserTokenExport{}, errors.New("missing auth_token")
	}
	return out, nil
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		switch v := values[key].(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func firstInt64(values map[string]any, keys ...string) int64 {
	for _, key := range keys {
		switch v := values[key].(type) {
		case float64:
			return int64(v)
		case string:
			n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if n > 0 {
				return n
			}
		}
	}
	return 0
}
