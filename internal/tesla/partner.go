package tesla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type partnerTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

// RegisterPartnerWithRetry waits for the public service to become reachable,
// then registers this app's domain with Tesla if it is not already registered.
func (a *App) RegisterPartnerWithRetry(ctx context.Context) error {
	var lastErr error
	for delay := time.Second; ; delay = min(delay*2, 15*time.Second) {
		if err := a.RegisterPartner(ctx); err == nil {
			return nil
		} else {
			lastErr = err
			log.Printf("Tesla partner registration attempt failed; retrying in %s: %v", delay, err)
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("register Tesla partner after retries: %w", lastErr)
		case <-timer.C:
		}
	}
}

// RegisterPartner registers the configured application domain if Tesla does
// not already have a public key for it in the configured Fleet API region.
func (a *App) RegisterPartner(ctx context.Context) error {
	domain := a.config.PartnerDomain
	if err := validateDomain(domain); err != nil {
		return err
	}

	partnerToken, err := a.partnerToken(ctx)
	if err != nil {
		return err
	}

	registered, err := a.partnerDomainRegistered(ctx, partnerToken, domain)
	if err != nil {
		return err
	}
	if registered {
		return nil
	}

	body, err := json.Marshal(struct {
		Domain string `json:"domain"`
	}{Domain: domain})
	if err != nil {
		return fmt.Errorf("encode Tesla partner registration: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.config.Audience, "/")+"/api/1/partner_accounts", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Tesla partner registration request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+partnerToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("register Tesla partner account: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return responseError("register Tesla partner account", response)
	}
	return nil
}

func (a *App) partnerToken(ctx context.Context) (string, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {a.config.OAuth.ClientID},
		"client_secret": {a.config.OAuth.ClientSecret},
		"audience":      {a.config.Audience},
		"scope":         {"openid vehicle_device_data vehicle_cmds vehicle_charging_cmds"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create Tesla partner-token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("request Tesla partner token: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", responseError("request Tesla partner token", response)
	}

	var token partnerTokenResponse
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil {
		return "", fmt.Errorf("decode Tesla partner token: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("Tesla returned a partner token response without an access token: %s", token.Error)
	}
	return token.AccessToken, nil
}

func (a *App) partnerDomainRegistered(ctx context.Context, partnerToken, domain string) (bool, error) {
	endpoint := strings.TrimRight(a.config.Audience, "/") + "/api/1/partner_accounts/public_key?" + url.Values{"domain": {domain}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("create Tesla partner-account lookup: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+partnerToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("look up Tesla partner account: %w", err)
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound, http.StatusPreconditionFailed:
		// Tesla can report an unregistered partner account as either 404 or
		// 412, depending on the Fleet API deployment.
		return false, nil
	default:
		return false, responseError("look up Tesla partner account", response)
	}
}

func validateDomain(domain string) error {
	if domain == "" || strings.Contains(domain, "://") || strings.Contains(domain, "/") || strings.Contains(domain, "?") {
		return fmt.Errorf("TESLA_PARTNER_DOMAIN must be a hostname without a scheme or path")
	}
	return nil
}

func responseError(action string, response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	return fmt.Errorf("%s: Tesla returned %s: %s", action, response.Status, strings.TrimSpace(string(body)))
}
