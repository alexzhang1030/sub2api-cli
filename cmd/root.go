package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/alex/sub2api-cli/internal/auth"
	"github.com/alex/sub2api-cli/internal/config"
	"github.com/alex/sub2api-cli/internal/render"
	"github.com/alex/sub2api-cli/internal/sub2api"
	"github.com/alex/sub2api-cli/internal/usage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type appOptions struct {
	configPath string
	profile    string
}

func NewRootCommand() *cobra.Command {
	opts := &appOptions{}
	cmd := &cobra.Command{
		Use:   "sub2api",
		Short: "sub2api usage dashboard CLI",
	}
	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&opts.profile, "profile", config.DefaultProfile, "profile name")
	cmd.AddCommand(newLoginCommand(opts), newTodayCommand(opts), newLogoutCommand(opts), newWhoamiCommand(opts))
	return cmd
}

func Execute() error {
	return NewRootCommand().Execute()
}

func newLoginCommand(opts *appOptions) *cobra.Command {
	var baseURL, provider, timezone string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login with browser OAuth",
		RunE: func(cmd *cobra.Command, args []string) error {
			if timezone == "" {
				timezone = config.LocalTimezone()
			}
			pair, err := auth.LoginWithBrowser(cmd.Context(), auth.OAuthOptions{
				BaseURL:  baseURL,
				Provider: provider,
				OnOpen: func(loginURL string, callbackURL string) {
					fmt.Fprintf(cmd.OutOrStdout(), "Opening browser for %s OAuth\n", provider)
					fmt.Fprintf(cmd.OutOrStdout(), "Local callback: %s\n", callbackURL)
					fmt.Fprintf(cmd.OutOrStdout(), "Login URL: %s\n", loginURL)
				},
			})
			if err != nil {
				return err
			}
			store, err := storeFromOptions(opts)
			if err != nil {
				return err
			}
			cfg, err := store.Load()
			if err != nil {
				return err
			}
			if cfg.Profiles == nil {
				cfg.Profiles = map[string]config.Profile{}
			}
			expiresAt := time.Time{}
			if pair.ExpiresIn > 0 {
				expiresAt = time.Now().Add(time.Duration(pair.ExpiresIn) * time.Second)
			}
			cfg.CurrentProfile = opts.profile
			cfg.Profiles[opts.profile] = config.Profile{
				Name:           opts.profile,
				BaseURL:        baseURL,
				Provider:       provider,
				Timezone:       timezone,
				TokenExpiresAt: expiresAt,
			}
			if err := auth.NewSystemKeychain().Set(opts.profile, pair); err != nil {
				return err
			}
			if err := store.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in profile %q\n", opts.profile)
			return nil
		},
	}
	cmd.Flags().StringVar(&baseURL, "base-url", "", "sub2api base URL")
	cmd.Flags().StringVar(&provider, "provider", "github", "OAuth provider: github, google, oidc, linuxdo, wechat")
	cmd.Flags().StringVar(&timezone, "timezone", config.LocalTimezone(), "IANA timezone")
	_ = cmd.MarkFlagRequired("base-url")
	cmd.AddCommand(newLoginTokenCommand(opts))
	return cmd
}

func newLoginTokenCommand(opts *appOptions) *cobra.Command {
	var baseURL, provider, timezone, fromFile string
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Import tokens copied from browser localStorage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if timezone == "" {
				timezone = config.LocalTimezone()
			}
			raw, err := readTokenInput(cmd, fromFile)
			if err != nil {
				return err
			}
			exported, err := auth.ParseBrowserTokenExport(raw)
			if err != nil {
				return err
			}
			store, err := storeFromOptions(opts)
			if err != nil {
				return err
			}
			cfg, err := store.Load()
			if err != nil {
				return err
			}
			if cfg.Profiles == nil {
				cfg.Profiles = map[string]config.Profile{}
			}
			expiresAt := time.Time{}
			if exported.ExpiresAtMS > 0 {
				expiresAt = time.UnixMilli(exported.ExpiresAtMS)
			}
			cfg.CurrentProfile = opts.profile
			cfg.Profiles[opts.profile] = config.Profile{
				Name:           opts.profile,
				BaseURL:        baseURL,
				Provider:       provider,
				Timezone:       timezone,
				TokenExpiresAt: expiresAt,
			}
			pair := sub2api.TokenPair{
				AccessToken:  exported.AccessToken,
				RefreshToken: exported.RefreshToken,
				TokenType:    "Bearer",
			}
			if err := auth.NewSystemKeychain().Set(opts.profile, pair); err != nil {
				return err
			}
			if err := store.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported browser tokens for profile %q\n", opts.profile)
			return nil
		},
	}
	cmd.Flags().StringVar(&baseURL, "base-url", "", "sub2api base URL")
	cmd.Flags().StringVar(&provider, "provider", "oidc", "provider name")
	cmd.Flags().StringVar(&timezone, "timezone", config.LocalTimezone(), "IANA timezone")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file exported from browser localStorage; reads stdin when empty")
	_ = cmd.MarkFlagRequired("base-url")
	return cmd
}

func readTokenInput(cmd *cobra.Command, fromFile string) ([]byte, error) {
	if strings.TrimSpace(fromFile) != "" {
		return os.ReadFile(fromFile)
	}
	return io.ReadAll(cmd.InOrStdin())
}

func newTodayCommand(opts *appOptions) *cobra.Command {
	const refreshEvery = 5 * time.Second
	cmd := &cobra.Command{
		Use:   "today",
		Short: "Render today's usage dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := loadTodayReport(cmd.Context(), opts)
			if err != nil {
				return err
			}
			program := tea.NewProgram(render.NewLiveDashboard(report, refreshEvery, func() (usage.Report, error) {
				return loadTodayReport(cmd.Context(), opts)
			}), tea.WithOutput(cmd.OutOrStdout()))
			_, err = program.Run()
			return err
		},
	}
	return cmd
}

func newWhoamiCommand(opts *appOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, client, err := clientFromOptions(cmd.Context(), opts)
			if err != nil {
				return err
			}
			user, err := client.GetCurrentUser(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s <%s> role=%s profile=%s base_url=%s\n", user.Username, user.Email, user.Role, profile.Name, profile.BaseURL)
			return nil
		},
	}
}

func newLogoutCommand(opts *appOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials for profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.NewSystemKeychain().Delete(opts.profile); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged out profile %q\n", opts.profile)
			return nil
		},
	}
}

func loadTodayReport(ctx context.Context, opts *appOptions) (usage.Report, error) {
	profile, client, err := clientFromOptions(ctx, opts)
	if err != nil {
		return usage.Report{}, err
	}
	now := time.Now()
	_, _, label, err := config.TodayRange(profile.Timezone, now)
	if err != nil {
		return usage.Report{}, err
	}
	stats, err := client.GetDashboardStats(ctx)
	if err != nil {
		return usage.Report{}, err
	}
	trend, err := client.GetDashboardTrend(ctx, label, label, "hour", profile.Timezone)
	if err != nil {
		return usage.Report{}, err
	}
	models, err := client.GetDashboardModels(ctx, label, label, profile.Timezone)
	if err != nil {
		return usage.Report{}, err
	}
	return usage.BuildTodayReport(usage.Input{
		Profile:   profile.Name,
		Date:      label,
		Timezone:  profile.Timezone,
		Generated: now,
		Stats:     stats,
		Trend:     trend.Trend,
		Models:    models.Models,
	}), nil
}

func clientFromOptions(ctx context.Context, opts *appOptions) (config.Profile, *sub2api.Client, error) {
	store, err := storeFromOptions(opts)
	if err != nil {
		return config.Profile{}, nil, err
	}
	cfg, err := store.Load()
	if err != nil {
		return config.Profile{}, nil, err
	}
	profile, ok := cfg.Profiles[opts.profile]
	if !ok && strings.TrimSpace(opts.profile) == config.DefaultProfile {
		profile, err = cfg.Current()
		if err != nil {
			return config.Profile{}, nil, err
		}
	} else if !ok {
		return config.Profile{}, nil, fmt.Errorf("profile %q not found; run sub2api login", opts.profile)
	}
	profile.Name = opts.profile
	pair, err := auth.NewSystemKeychain().Get(profile.Name)
	if err != nil {
		return config.Profile{}, nil, err
	}
	client := sub2api.NewClient(profile.BaseURL, pair.AccessToken, &sub2api.RefreshOptions{
		RefreshToken: pair.RefreshToken,
		OnRefresh: func(newPair sub2api.TokenPair) error {
			if err := auth.NewSystemKeychain().Set(profile.Name, newPair); err != nil {
				return err
			}
			if newPair.ExpiresIn > 0 {
				profile.TokenExpiresAt = time.Now().Add(time.Duration(newPair.ExpiresIn) * time.Second)
				cfg.Profiles[profile.Name] = profile
				return store.Save(cfg)
			}
			return nil
		},
	})
	_ = ctx
	return profile, client, nil
}

func storeFromOptions(opts *appOptions) (*config.Store, error) {
	path := opts.configPath
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = config.DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	return config.NewStore(path), nil
}

func ExitOnError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
