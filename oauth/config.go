package oauth

import (
	"encoding/base64"

	"github.com/blend/go-sdk/env"
)

// Config is the config options.
type Config struct {
	// Secret is an encryption key used to verify oauth state.
	Secret string `json:"secret,omitempty" yaml:"secret,omitempty" env:"OAUTH_SECRET"`
	// RedirectURI is the oauth return url.
	RedirectURI string `json:"redirectURI,omitempty" yaml:"redirectURI,omitempty" env:"OAUTH_REDIRECT_URI"`
	// HostedDomain is a specific domain we want to filter identities to.
	HostedDomain string `json:"hostedDomain,omitempty" yaml:"hostedDomain,omitempty" env:"OAUTH_HOSTED_DOMAIN"`
	// Scopes are oauth scopes to request.
	Scopes []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	// ClientID is part of the oauth credential pair.
	ClientID string `json:"clientID,omitempty" yaml:"clientID,omitempty" env:"OAUTH_CLIENT_ID"`
	// ClientSecret is part of the oauth credential pair.
	ClientSecret string `json:"clientSecret,omitempty" yaml:"clientSecret,omitempty" env:"OAUTH_CLIENT_SECRET"`
}

// IsZero returns if the config is set or not.
func (c Config) IsZero() bool {
	return len(c.ClientID) == 0 || len(c.ClientSecret) == 0
}

// Resolve adds extra steps to perform during `configutil.Read(...)`.
func (c *Config) Resolve() error {
	return env.Env().ReadInto(c)
}

// DecodeSecret decodes the secret if set from base64 encoding.
func (c Config) DecodeSecret() ([]byte, error) {
	if len(c.Secret) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(c.Secret)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}
	return nil, nil
}

// ScopesOrDefault gets oauth scopes to authenticate with or a default set of scopes.
func (c Config) ScopesOrDefault() []string {
	if len(c.Scopes) > 0 {
		return c.Scopes
	}
	return DefaultScopes
}
