package sub2api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RefreshOptions struct {
	RefreshToken string
	OnRefresh    func(TokenPair) error
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
	refresh *RefreshOptions
}

func NewClient(baseURL string, token string, refresh *RefreshOptions) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
		refresh: refresh,
	}
}

func (c *Client) GetDashboardStats(ctx context.Context) (DashboardStats, error) {
	var out DashboardStats
	err := c.get(ctx, "/api/v1/usage/dashboard/stats", nil, &out)
	return out, err
}

func (c *Client) GetDashboardTrend(ctx context.Context, startDate, endDate, granularity, timezone string) (TrendResponse, error) {
	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)
	params.Set("granularity", granularity)
	params.Set("timezone", timezone)
	var out TrendResponse
	err := c.get(ctx, "/api/v1/usage/dashboard/trend", params, &out)
	return out, err
}

func (c *Client) GetDashboardModels(ctx context.Context, startDate, endDate, timezone string) (ModelsResponse, error) {
	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)
	params.Set("timezone", timezone)
	var out ModelsResponse
	err := c.get(ctx, "/api/v1/usage/dashboard/models", params, &out)
	return out, err
}

func (c *Client) GetCurrentUser(ctx context.Context) (CurrentUser, error) {
	var out CurrentUser
	err := c.get(ctx, "/api/v1/auth/me", nil, &out)
	return out, err
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	var out TokenPair
	body := map[string]string{"refresh_token": refreshToken}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/refresh", nil, body, &out, false); err != nil {
		return TokenPair{}, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return TokenPair{}, errors.New("sub2api: refresh response missing access_token")
	}
	c.token = out.AccessToken
	return out, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, params, nil, out, true)
}

func (c *Client) doJSON(ctx context.Context, method string, path string, params url.Values, body any, out any, allowRefresh bool) error {
	err := c.doJSONOnce(ctx, method, path, params, body, out)
	if allowRefresh && isUnauthorized(err) && c.refresh != nil && strings.TrimSpace(c.refresh.RefreshToken) != "" {
		pair, refreshErr := c.Refresh(ctx, c.refresh.RefreshToken)
		if refreshErr != nil {
			return refreshErr
		}
		if c.refresh.OnRefresh != nil {
			if saveErr := c.refresh.OnRefresh(pair); saveErr != nil {
				return saveErr
			}
		}
		return c.doJSONOnce(ctx, method, path, params, body, out)
	}
	return err
}

func (c *Client) doJSONOnce(ctx context.Context, method string, path string, params url.Values, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	endpoint := c.baseURL + path
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return apiError{status: resp.StatusCode, message: readAPIMessage(raw, "unauthorized")}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError{status: resp.StatusCode, message: readAPIMessage(raw, resp.Status)}
	}
	return unwrap(raw, out)
}

func unwrap(raw []byte, out any) error {
	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		return apiError{status: envelope.Code, message: envelope.Message}
	}
	if out == nil {
		return nil
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}

type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string {
	if strings.TrimSpace(e.message) == "" {
		return fmt.Sprintf("sub2api: request failed (%d)", e.status)
	}
	return "sub2api: " + e.message
}

func isUnauthorized(err error) bool {
	var apiErr apiError
	return errors.As(err, &apiErr) && apiErr.status == http.StatusUnauthorized
}

func readAPIMessage(raw []byte, fallback string) string {
	var envelope struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		if strings.TrimSpace(envelope.Message) != "" {
			return envelope.Message
		}
		if strings.TrimSpace(envelope.Detail) != "" {
			return envelope.Detail
		}
	}
	return fallback
}
