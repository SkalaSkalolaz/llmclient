package llmclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type AudioRequest struct {
	Provider string
	Model    string
	APIKey   string
	Prompt   string
}

type AudioResponse struct {
	Data []byte
}

func (c *Client) GenerateAudio(ctx context.Context, req *AudioRequest) (*AudioResponse, error) {
	if req == nil {
		return nil, errors.New("audio request is nil")
	}

	provider, err := c.newAudioProvider(req)
	if err != nil {
		return nil, err
	}

	data, err := provider.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	return &AudioResponse{Data: data}, nil
}

func (c *Client) newAudioProvider(req *AudioRequest) (audioProvider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "pollinations":
		return &pollinationsAudioProvider{client: c.httpClient}, nil
	default:
		return nil, fmt.Errorf("unknown audio provider: %s", req.Provider)
	}
}

type audioProvider interface {
	Generate(ctx context.Context, req *AudioRequest) ([]byte, error)
}

type pollinationsAudioProvider struct {
	client *http.Client
}

func (p *pollinationsAudioProvider) Generate(ctx context.Context, req *AudioRequest) ([]byte, error) {
	encodedPrompt := url.PathEscape(req.Prompt)
	endpoint := fmt.Sprintf("https://gen.pollinations.ai/audio/%s", encodedPrompt)

	params := url.Values{}
	if req.Model != "" {
		params.Set("model", req.Model)
	}

	if len(params) > 0 {
		endpoint = endpoint + "?" + params.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}
