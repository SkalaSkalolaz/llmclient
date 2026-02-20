package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Profile struct {
	ID           string         `json:"id,omitempty"`
	Email        string         `json:"email,omitempty"`
	Name         string         `json:"name,omitempty"`
	Username     string         `json:"username,omitempty"`
	Credits      float64        `json:"credits,omitempty"`
	Balance      float64        `json:"balance,omitempty"`
	Usage        *ProfileUsage  `json:"usage,omitempty"`
	Limits       *ProfileLimits `json:"limits,omitempty"`
	CreatedAt    string         `json:"created_at,omitempty"`
	Plan         string         `json:"plan,omitempty"`
	Subscription string         `json:"subscription,omitempty"`
	Raw          map[string]any `json:"-"`
}

type ProfileUsage struct {
	TotalTokens      int64   `json:"total_tokens,omitempty"`
	PromptTokens     int64   `json:"prompt_tokens,omitempty"`
	CompletionTokens int64   `json:"completion_tokens,omitempty"`
	TotalRequests    int64   `json:"total_requests,omitempty"`
	TotalCost        float64 `json:"total_cost,omitempty"`
	PeriodStart      string  `json:"period_start,omitempty"`
	PeriodEnd        string  `json:"period_end,omitempty"`
}

type ProfileLimits struct {
	RequestsPerDay int64 `json:"requests_per_day,omitempty"`
	TokensPerDay   int64 `json:"tokens_per_day,omitempty"`
	TokensPerMonth int64 `json:"tokens_per_month,omitempty"`
	RequestsUsed   int64 `json:"requests_used,omitempty"`
	TokensUsed     int64 `json:"tokens_used,omitempty"`
}

type ProfileResponse struct {
	Profile *Profile
	Raw     []byte
}

type ProfileRequest struct {
	Provider string
	APIKey   string
}

type profileProvider interface {
	GetProfile(ctx context.Context, req *ProfileRequest) (*Profile, []byte, error)
}

type profileProviderFactory func(*http.Client) profileProvider

var registeredProfileProviders = make(map[string]profileProviderFactory)

func RegisterProfileProvider(name string, factory profileProviderFactory) {
	registeredProfileProviders[strings.ToLower(name)] = factory
}

func (c *Client) GetProfile(ctx context.Context, req *ProfileRequest) (*ProfileResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("profile request is nil")
	}

	provider, err := c.newProfileProvider(req)
	if err != nil {
		return nil, err
	}

	profile, raw, err := provider.GetProfile(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ProfileResponse{Profile: profile, Raw: raw}, nil
}

func (c *Client) newProfileProvider(req *ProfileRequest) (profileProvider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "pollinations":
		return &pollinationsProfileProvider{client: c.httpClient}, nil
	default:
		if custom, ok := registeredProfileProviders[name]; ok {
			return custom(c.httpClient), nil
		}
		return nil, fmt.Errorf("unknown profile provider: %s", req.Provider)
	}
}

type pollinationsProfileProvider struct {
	client *http.Client
}

func (p *pollinationsProfileProvider) GetProfile(ctx context.Context, req *ProfileRequest) (*Profile, []byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", "https://gen.pollinations.ai/account/profile", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(data))
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, nil, fmt.Errorf("parse response: %w", err)
	}

	profile.Raw = make(map[string]any)
	_ = json.Unmarshal(data, &profile.Raw)

	return &profile, data, nil
}

func GetProfile(provider, apiKey string) (*Profile, error) {
	return GetProfileWithContext(context.Background(), provider, apiKey)
}

func GetProfileWithContext(ctx context.Context, provider, apiKey string) (*Profile, error) {
	client := NewClient()
	resp, err := client.GetProfile(ctx, &ProfileRequest{
		Provider: provider,
		APIKey:   apiKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.Profile, nil
}

func (p *Profile) HasCredits() bool {
	return p.Credits > 0 || p.Balance > 0
}

func (p *Profile) UsagePercent() float64 {
	if p.Limits == nil || p.Limits.TokensPerMonth == 0 {
		return 0
	}
	return float64(p.Limits.TokensUsed) / float64(p.Limits.TokensPerMonth) * 100
}
