package auth

import (
	"errors"
	"strings"

	"github.com/alex/sub2api-cli/internal/sub2api"
	"github.com/zalando/go-keyring"
)

const serviceName = "sub2api-cli"

type Keychain interface {
	Set(profile string, pair sub2api.TokenPair) error
	Get(profile string) (sub2api.TokenPair, error)
	Delete(profile string) error
}

type SystemKeychain struct{}

func NewSystemKeychain() SystemKeychain {
	return SystemKeychain{}
}

func (SystemKeychain) Set(profile string, pair sub2api.TokenPair) error {
	if strings.TrimSpace(profile) == "" {
		return errors.New("profile is empty")
	}
	if err := keyring.Set(serviceName, profile+":access_token", pair.AccessToken); err != nil {
		return err
	}
	if err := keyring.Set(serviceName, profile+":refresh_token", pair.RefreshToken); err != nil {
		return err
	}
	return nil
}

func (SystemKeychain) Get(profile string) (sub2api.TokenPair, error) {
	access, err := keyring.Get(serviceName, profile+":access_token")
	if err != nil {
		return sub2api.TokenPair{}, err
	}
	refresh, _ := keyring.Get(serviceName, profile+":refresh_token")
	return sub2api.TokenPair{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer"}, nil
}

func (SystemKeychain) Delete(profile string) error {
	_ = keyring.Delete(serviceName, profile+":access_token")
	_ = keyring.Delete(serviceName, profile+":refresh_token")
	return nil
}
