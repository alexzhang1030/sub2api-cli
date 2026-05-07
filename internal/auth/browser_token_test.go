package auth

import "testing"

func TestParseBrowserTokenExport(t *testing.T) {
	raw := []byte(`{
		"auth_token": "access-1",
		"refresh_token": "refresh-1",
		"token_expires_at": "1778101200000"
	}`)

	out, err := ParseBrowserTokenExport(raw)
	if err != nil {
		t.Fatalf("parse export: %v", err)
	}
	if out.AccessToken != "access-1" {
		t.Fatalf("access token = %q", out.AccessToken)
	}
	if out.RefreshToken != "refresh-1" {
		t.Fatalf("refresh token = %q", out.RefreshToken)
	}
	if out.ExpiresAtMS != 1778101200000 {
		t.Fatalf("expires at = %d", out.ExpiresAtMS)
	}
}

func TestParseBrowserTokenExportAcceptsFrontendNames(t *testing.T) {
	raw := []byte(`{"access_token":"access-2","refresh_token":"refresh-2","expires_at":1778101200000}`)

	out, err := ParseBrowserTokenExport(raw)
	if err != nil {
		t.Fatalf("parse export: %v", err)
	}
	if out.AccessToken != "access-2" || out.RefreshToken != "refresh-2" {
		t.Fatalf("tokens = %+v", out)
	}
}
