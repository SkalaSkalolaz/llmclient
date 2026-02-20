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

type ModelPricing struct {
	Currency              string  `json:"currency"`
	PromptTextTokens      float64 `json:"promptTextTokens,omitempty"`
	PromptCachedTokens    float64 `json:"promptCachedTokens,omitempty"`
	PromptAudioTokens     float64 `json:"promptAudioTokens,omitempty"`
	CompletionTextTokens  float64 `json:"completionTextTokens,omitempty"`
	CompletionAudioTokens float64 `json:"completionAudioTokens,omitempty"`
}

type Model struct {
	Name             string         `json:"name"`
	Aliases          []string       `json:"aliases,omitempty"`
	Description      string         `json:"description,omitempty"`
	Pricing          *ModelPricing  `json:"pricing,omitempty"`
	InputModalities  []string       `json:"input_modalities,omitempty"`
	OutputModalities []string       `json:"output_modalities,omitempty"`
	Tools            bool           `json:"tools,omitempty"`
	Reasoning        bool           `json:"reasoning,omitempty"`
	IsSpecialized    bool           `json:"is_specialized,omitempty"`
	PaidOnly         bool           `json:"paid_only,omitempty"`
	ContextWindow    int            `json:"context_window,omitempty"`
	Voices           []string       `json:"voices,omitempty"`
	Raw              map[string]any `json:"-"`
}

type ModelsResponse struct {
	Models []Model `json:"models"`
	Raw    []byte  `json:"-"`
}

type ModelsRequest struct {
	Provider string
	APIKey   string
}

func (c *Client) ListTextModels(ctx context.Context, req *ModelsRequest) (*ModelsResponse, error) {
	if req == nil {
		return nil, errors.New("models request is nil")
	}

	provider, err := c.newModelsProvider(req)
	if err != nil {
		return nil, err
	}

	models, raw, err := provider.ListModels(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ModelsResponse{Models: models, Raw: raw}, nil
}

func (c *Client) newModelsProvider(req *ModelsRequest) (modelsProvider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "pollinations":
		return &pollinationsModelsProvider{client: c.httpClient}, nil
	default:
		if custom, ok := registeredModelsProviders[name]; ok {
			return custom(c.httpClient), nil
		}
		return nil, fmt.Errorf("unknown models provider: %s", req.Provider)
	}
}

type modelsProvider interface {
	ListModels(ctx context.Context, req *ModelsRequest) ([]Model, []byte, error)
}

type modelsProviderFactory func(*http.Client) modelsProvider

var registeredModelsProviders = make(map[string]modelsProviderFactory)

func RegisterModelsProvider(name string, factory modelsProviderFactory) {
	registeredModelsProviders[strings.ToLower(name)] = factory
}

type pollinationsModelsProvider struct {
	client *http.Client
}

func (p *pollinationsModelsProvider) ListModels(ctx context.Context, req *ModelsRequest) ([]Model, []byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", "https://gen.pollinations.ai/text/models", nil)
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

	var models []Model
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, nil, fmt.Errorf("parse response: %w", err)
	}

	for i := range models {
		models[i].Raw = make(map[string]any)
		_ = json.Unmarshal(data, &models[i].Raw)
	}

	return models, data, nil
}

func ListTextModels(provider, apiKey string) ([]Model, error) {
	return ListTextModelsWithContext(context.Background(), provider, apiKey)
}

func ListTextModelsWithContext(ctx context.Context, provider, apiKey string) ([]Model, error) {
	client := NewClient()
	resp, err := client.ListTextModels(ctx, &ModelsRequest{
		Provider: provider,
		APIKey:   apiKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.Models, nil
}

func (m *Model) HasInputModality(modality string) bool {
	for _, mod := range m.InputModalities {
		if mod == modality {
			return true
		}
	}
	return false
}

func (m *Model) HasOutputModality(modality string) bool {
	for _, mod := range m.OutputModalities {
		if mod == modality {
			return true
		}
	}
	return false
}

func (m *Model) HasAlias(alias string) bool {
	for _, a := range m.Aliases {
		if a == alias {
			return true
		}
	}
	return false
}

func (m *Model) EffectivePricePer1kTokens() float64 {
	if m.Pricing == nil {
		return 0
	}
	return (m.Pricing.PromptTextTokens + m.Pricing.CompletionTextTokens) * 1000
}

func FilterModelsByModality(models []Model, inputModality, outputModality string) []Model {
	var result []Model
	for _, m := range models {
		if inputModality != "" && !m.HasInputModality(inputModality) {
			continue
		}
		if outputModality != "" && !m.HasOutputModality(outputModality) {
			continue
		}
		result = append(result, m)
	}
	return result
}

func FilterModelsByCapability(models []Model, tools, reasoning bool) []Model {
	var result []Model
	for _, m := range models {
		if tools && !m.Tools {
			continue
		}
		if reasoning && !m.Reasoning {
			continue
		}
		result = append(result, m)
	}
	return result
}

func FilterFreeModels(models []Model) []Model {
	var result []Model
	for _, m := range models {
		if !m.PaidOnly {
			result = append(result, m)
		}
	}
	return result
}
