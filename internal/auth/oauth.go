package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alex/sub2api-cli/internal/sub2api"
)

type OAuthOptions struct {
	BaseURL  string
	Provider string
	Timeout  time.Duration
	OnOpen   func(loginURL string, callbackURL string)
}

func LoginWithBrowser(ctx context.Context, opts OAuthOptions) (sub2api.TokenPair, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return sub2api.TokenPair{}, err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := "http://127.0.0.1:" + strconv.Itoa(port) + "/callback"
	tokenCh := make(chan sub2api.TokenPair, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		pair, err := tokenPairFromValues(q)
		if err == nil {
			_, _ = fmt.Fprint(w, "sub2api CLI login complete. You can close this tab.")
			tokenCh <- pair
			return
		}
		_, _ = fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><script>
const raw = window.location.hash.startsWith("#") ? window.location.hash.slice(1) : window.location.hash;
const target = new URL(window.location.href);
target.hash = "";
target.search = raw;
window.location.replace(target.toString());
</script><body>Completing sub2api CLI login...</body>`)
	})
	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()
	defer server.Shutdown(context.Background())

	loginURL, err := buildStartURL(opts.BaseURL, opts.Provider, callbackURL)
	if err != nil {
		return sub2api.TokenPair{}, err
	}
	if opts.OnOpen != nil {
		opts.OnOpen(loginURL, callbackURL)
	}
	if err := OpenBrowser(loginURL); err != nil {
		return sub2api.TokenPair{}, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	select {
	case pair := <-tokenCh:
		return pair, nil
	case err := <-errCh:
		return sub2api.TokenPair{}, err
	case <-waitCtx.Done():
		return sub2api.TokenPair{}, loginTimeoutError(opts.Provider, callbackURL)
	}
}

func loginTimeoutError(provider string, callbackURL string) error {
	if provider == "oidc" || provider == "linuxdo" || provider == "wechat" {
		return fmt.Errorf("%s login timed out: the browser login completed, but the CLI did not receive tokens on the local callback %s. This provider uses Sub2API pending exchange; configure the provider frontend redirect URL to the local callback for this run, or add a Sub2API CLI token exchange endpoint that can return access_token and refresh_token to the CLI", strings.ToUpper(provider), callbackURL)
	}
	return fmt.Errorf("%s login timed out: the CLI did not receive access_token on the local callback %s. Configure the provider frontend redirect URL to the local callback for this run", strings.ToUpper(provider), callbackURL)
}

func buildStartURL(baseURL, provider, callbackURL string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", errors.New("base url is required")
	}
	switch provider {
	case "github", "google", "oidc", "linuxdo", "wechat":
	default:
		return "", fmt.Errorf("unsupported provider %q", provider)
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/v1/auth/oauth/" + provider + "/start")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("redirect", callbackURL)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func tokenPairFromValues(values url.Values) (sub2api.TokenPair, error) {
	if msg := strings.TrimSpace(values.Get("error_description")); msg != "" {
		return sub2api.TokenPair{}, errors.New(msg)
	}
	if msg := strings.TrimSpace(values.Get("error")); msg != "" {
		return sub2api.TokenPair{}, errors.New(msg)
	}
	access := strings.TrimSpace(values.Get("access_token"))
	if access == "" {
		return sub2api.TokenPair{}, errors.New("missing access_token")
	}
	expiresIn, _ := strconv.Atoi(strings.TrimSpace(values.Get("expires_in")))
	tokenType := strings.TrimSpace(values.Get("token_type"))
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return sub2api.TokenPair{
		AccessToken:  access,
		RefreshToken: strings.TrimSpace(values.Get("refresh_token")),
		ExpiresIn:    expiresIn,
		TokenType:    tokenType,
	}, nil
}

func OpenBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}
