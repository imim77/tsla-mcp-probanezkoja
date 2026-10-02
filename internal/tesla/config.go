package tesla

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
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
	Audience        string
	OAuth           oauth2.Config
	LogTokens       bool
	RegisterPartner bool
	PartnerDomain   string
	PublicKeyPEM    []byte
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

	config := Config{
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
	}

	privateKeyPath := os.Getenv("TESLA_PRIVATE_KEY_FILE")
	config.RegisterPartner = os.Getenv("TESLA_REGISTER_PARTNER") == "true"
	if config.RegisterPartner {
		config.PartnerDomain, err = requiredEnv("TESLA_PARTNER_DOMAIN")
		if err != nil {
			return Config{}, err
		}
		if privateKeyPath == "" {
			privateKeyPath = "/etc/secrets/tesla-private-key.pem"
		}
	}
	if privateKeyPath != "" {
		config.PublicKeyPEM, err = publicKeyFromPrivateKeyFile(privateKeyPath)
		if err != nil {
			return Config{}, err
		}
	}
	if config.RegisterPartner && len(config.PublicKeyPEM) == 0 {
		return Config{}, fmt.Errorf("TESLA_PRIVATE_KEY_FILE is required when TESLA_REGISTER_PARTNER is true")
	}

	return config, nil
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("missing environment variable %s", key)
	}
	return value, nil
}

func publicKeyFromPrivateKeyFile(path string) ([]byte, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Tesla private key: %w", err)
	}
	block, _ := pem.Decode(contents)
	if block == nil {
		return nil, fmt.Errorf("Tesla private key is not PEM encoded")
	}

	var privateKey *ecdsa.PrivateKey
	switch block.Type {
	case "EC PRIVATE KEY":
		privateKey, err = x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			err = parseErr
			break
		}
		var ok bool
		privateKey, ok = parsed.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("Tesla private key must be an EC private key")
		}
	default:
		return nil, fmt.Errorf("unsupported Tesla private-key PEM type %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("parse Tesla private key: %w", err)
	}
	if privateKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("Tesla private key must use the prime256v1 curve")
	}

	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("encode Tesla public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey}), nil
}
