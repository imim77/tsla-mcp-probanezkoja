package tesla

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

const (
	authURL  = "https://auth.tesla.com/oauth2/v3/authorize"
	tokenURL = "https://fleet-auth.prd.vn.cloud.tesla.com/oauth2/v3/token"
)

// Config contains the Tesla OAuth settings supplied by the environment.
type Config struct {
	Audience  string
	OAuth     oauth2.Config
	LogTokens bool
}

// LoadConfigFromEnv builds configuration from required Tesla OAuth environment variables.
func LoadConfigFromEnv() (Config, error) {
	clientID, err := requiredEnv("TESLA_CLIENT_ID")
	if err != nil {
		return Config{}, err
	}
	clientSecret, err := requiredEnv("TESLA_CLIENT_SECRET")
	if err != nil {
		return Config{}, err
	}
	redirectURL, err := requiredEnv("TESLA_REDIRECT_URI")
	if err != nil {
		return Config{}, err
	}
	audience, err := requiredEnv("TESLA_AUDIENCE")
	if err != nil {
		return Config{}, err
	}

	return Config{
		Audience:  audience,
		LogTokens: os.Getenv("TESLA_LOG_TOKENS") == "true",
		OAuth: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"openid", "offline_access", "user_data", "vehicle_device_data", "vehicle_cmds", "vehicle_charging_cmds",
			},
			Endpoint: oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL, AuthStyle: oauth2.AuthStyleInParams},
		},
	}, nil
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("missing environment variable %s", key)
	}
	return value, nil
}
