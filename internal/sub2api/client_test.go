package sub2api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientUnwrapsStandardResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/usage/dashboard/stats" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"today_tokens":42,"today_actual_cost":1.25}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", nil)
	stats, err := client.GetDashboardStats(context.Background())
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.TodayTokens != 42 {
		t.Fatalf("today tokens = %d", stats.TodayTokens)
	}
	if stats.TodayActualCost != 1.25 {
		t.Fatalf("today actual cost = %f", stats.TodayActualCost)
	}
}

func TestClientRefreshesTokenAndRetriesOnUnauthorized(t *testing.T) {
	var usageCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/usage/dashboard/stats":
			usageCalls++
			if r.Header.Get("Authorization") != "Bearer new-token" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":401,"message":"expired"}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"today_tokens":99}}`))
		case "/api/v1/auth/refresh":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"access_token":"new-token","refresh_token":"new-refresh","expires_in":3600,"token_type":"Bearer"}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var saved TokenPair
	client := NewClient(server.URL, "old-token", &RefreshOptions{
		RefreshToken: "refresh",
		OnRefresh: func(pair TokenPair) error {
			saved = pair
			return nil
		},
	})
	stats, err := client.GetDashboardStats(context.Background())
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.TodayTokens != 99 {
		t.Fatalf("today tokens = %d", stats.TodayTokens)
	}
	if usageCalls != 2 {
		t.Fatalf("usage calls = %d", usageCalls)
	}
	if saved.AccessToken != "new-token" || saved.RefreshToken != "new-refresh" {
		t.Fatalf("saved pair = %+v", saved)
	}
}

func TestAPIErrorIncludesMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":400,"message":"bad date"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", nil)
	_, err := client.GetDashboardStats(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "sub2api: bad date" {
		t.Fatalf("error = %q", err.Error())
	}
}
