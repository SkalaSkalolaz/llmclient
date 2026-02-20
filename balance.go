package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Balance struct {
	Credits  float64        `json:"credits,omitempty"`
	Balance  float64        `json:"balance,omitempty"`
	Currency string         `json:"currency,omitempty"`
	Raw      map[string]any `json:"-"`
}

type BalanceResponse struct {
	Balance *Balance
	Raw     []byte
}

type BalanceRequest struct {
	Provider string
	APIKey   string
}

type balanceProvider interface {
	GetBalance(ctx context.Context, req *BalanceRequest) (*Balance, []byte, error)
}

type balanceProviderFactory func(*http.Client) balanceProvider

var registeredBalanceProviders = make(map[string]balanceProviderFactory)

func RegisterBalanceProvider(name string, factory balanceProviderFactory) {
	registeredBalanceProviders[strings.ToLower(name)] = factory
}

func (c *Client) GetBalance(ctx context.Context, req *BalanceRequest) (*BalanceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("balance request is nil")
	}

	provider, err := c.newBalanceProvider(req)
	if err != nil {
		return nil, err
	}

	bal, raw, err := provider.GetBalance(ctx, req)
	if err != nil {
		return nil, err
	}

	return &BalanceResponse{Balance: bal, Raw: raw}, nil
}

func (c *Client) newBalanceProvider(req *BalanceRequest) (balanceProvider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "pollinations":
		return &pollinationsBalanceProvider{client: c.httpClient}, nil
	default:
		if custom, ok := registeredBalanceProviders[name]; ok {
			return custom(c.httpClient), nil
		}
		return nil, fmt.Errorf("unknown balance provider: %s", req.Provider)
	}
}

type pollinationsBalanceProvider struct {
	client *http.Client
}

func (p *pollinationsBalanceProvider) GetBalance(ctx context.Context, req *BalanceRequest) (*Balance, []byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", "https://gen.pollinations.ai/account/balance", nil)
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

	var balance Balance
	if err := json.Unmarshal(data, &balance); err != nil {
		return nil, nil, fmt.Errorf("parse response: %w", err)
	}

	balance.Raw = make(map[string]any)
	_ = json.Unmarshal(data, &balance.Raw)

	return &balance, data, nil
}

func GetBalance(provider, apiKey string) (*Balance, error) {
	return GetBalanceWithContext(context.Background(), provider, apiKey)
}

func GetBalanceWithContext(ctx context.Context, provider, apiKey string) (*Balance, error) {
	client := NewClient()
	resp, err := client.GetBalance(ctx, &BalanceRequest{Provider: provider, APIKey: apiKey})
	if err != nil {
		return nil, err
	}
	return resp.Balance, nil
}

func (b *Balance) HasCredits() bool {
	return b.Credits > 0 || b.Balance > 0
}
