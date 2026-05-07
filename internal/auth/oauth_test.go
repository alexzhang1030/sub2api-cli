package auth

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildStartURLUsesProvidedCallbackPath(t *testing.T) {
	got, err := buildStartURL("https://dev.ai.sr/", "github", "http://127.0.0.1:14545/callback")
	if err != nil {
		t.Fatalf("build start url: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if u.String() != "https://dev.ai.sr/api/v1/auth/oauth/github/start?redirect=http%3A%2F%2F127.0.0.1%3A14545%2Fcallback" {
		t.Fatalf("url = %s", u.String())
	}
}

func TestOIDCTimeoutErrorExplainsPendingExchangeConstraint(t *testing.T) {
	err := loginTimeoutError("oidc", "http://127.0.0.1:14545/callback")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"OIDC", "pending exchange", "local callback"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message %q missing %q", msg, want)
		}
	}
}
