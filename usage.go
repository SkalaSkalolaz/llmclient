package llmclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type UsageFormat string

const (
	UsageFormatJSON UsageFormat = "json"
	UsageFormatCSV  UsageFormat = "csv"
)

type UsageRequest struct {
	Provider string
	APIKey   string
	Format   UsageFormat
}

type UsageRecord struct {
	Timestamp string         `json:"timestamp,omitempty"`
	Model     string         `json:"model,omitempty"`
	Provider  string         `json:"provider,omitempty"`
	Type      string         `json:"type,omitempty"`
	Prompt    string         `json:"prompt,omitempty"`
	Tokens    int64          `json:"tokens,omitempty"`
	Cost      float64        `json:"cost,omitempty"`
	Currency  string         `json:"currency,omitempty"`
	Raw       map[string]any `json:"-"`
}

type UsageTotals struct {
	TotalRequests int64   `json:"total_requests,omitempty"`
	TotalTokens   int64   `json:"total_tokens,omitempty"`
	TotalCost     float64 `json:"total_cost,omitempty"`
	Currency      string  `json:"currency,omitempty"`
}

type Usage struct {
	Records []UsageRecord `json:"records,omitempty"`
	Totals  *UsageTotals  `json:"totals,omitempty"`
	Raw     map[string]any
}

type UsageResponse struct {
	Usage *Usage
	Raw   []byte
}

type usageProvider interface {
	GetUsage(ctx context.Context, req *UsageRequest) (*Usage, []byte, error)
}

type usageProviderFactory func(*http.Client) usageProvider

var registeredUsageProviders = make(map[string]usageProviderFactory)

func RegisterUsageProvider(name string, factory usageProviderFactory) {
	registeredUsageProviders[strings.ToLower(name)] = factory
}

func (c *Client) GetUsage(ctx context.Context, req *UsageRequest) (*UsageResponse, error) {
	if req == nil {
		return nil, errors.New("usage request is nil")
	}
	if req.Format == "" {
		req.Format = UsageFormatJSON
	}

	provider, err := c.newUsageProvider(req)
	if err != nil {
		return nil, err
	}

	usage, raw, err := provider.GetUsage(ctx, req)
	if err != nil {
		return nil, err
	}

	return &UsageResponse{Usage: usage, Raw: raw}, nil
}

func (c *Client) newUsageProvider(req *UsageRequest) (usageProvider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "pollinations":
		return &pollinationsUsageProvider{client: c.httpClient}, nil
	default:
		if custom, ok := registeredUsageProviders[name]; ok {
			return custom(c.httpClient), nil
		}
		return nil, fmt.Errorf("unknown usage provider: %s", req.Provider)
	}
}

type pollinationsUsageProvider struct {
	client *http.Client
}

func (p *pollinationsUsageProvider) GetUsage(ctx context.Context, req *UsageRequest) (*Usage, []byte, error) {
	url := "https://gen.pollinations.ai/account/usage"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}

	switch req.Format {
	case UsageFormatJSON:
	case UsageFormatCSV:
		httpReq.Header.Set("Accept", "text/csv")
	default:
		return nil, nil, fmt.Errorf("unsupported usage format: %s", req.Format)
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

	if req.Format == UsageFormatCSV {
		return &Usage{Raw: map[string]any{"csv": string(data)}}, data, nil
	}

	var usage Usage
	if err := json.Unmarshal(data, &usage); err != nil {
		var alt struct {
			Usage *Usage `json:"usage"`
			Data  *Usage `json:"data"`
		}
		if err2 := json.Unmarshal(data, &alt); err2 == nil {
			if alt.Usage != nil {
				usage = *alt.Usage
			} else if alt.Data != nil {
				usage = *alt.Data
			} else {
				return nil, nil, fmt.Errorf("parse response: %w", err)
			}
		} else {
			return nil, nil, fmt.Errorf("parse response: %w", err)
		}
	}

	usage.Raw = make(map[string]any)
	_ = json.Unmarshal(data, &usage.Raw)

	return &usage, data, nil
}

func GetUsage(provider, apiKey string, format UsageFormat) (*Usage, error) {
	return GetUsageWithContext(context.Background(), provider, apiKey, format)
}

func GetUsageWithContext(ctx context.Context, provider, apiKey string, format UsageFormat) (*Usage, error) {
	client := NewClient()
	resp, err := client.GetUsage(ctx, &UsageRequest{Provider: provider, APIKey: apiKey, Format: format})
	if err != nil {
		return nil, err
	}
	return resp.Usage, nil
}
