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

type ImageRequest struct {
	Provider string
	Model    string
	APIKey   string
	Prompt   string
	Width    *int
	Height   *int
	Seed     *int
}

type ImageResponse struct {
	Data []byte
}

func (c *Client) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	if req == nil {
		return nil, errors.New("image request is nil")
	}

	provider, err := c.newImageProvider(req)
	if err != nil {
		return nil, err
	}

	data, err := provider.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ImageResponse{Data: data}, nil
}

func (c *Client) newImageProvider(req *ImageRequest) (imageProvider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "pollinations":
		return &pollinationsImageProvider{client: c.httpClient}, nil
	default:
		return nil, fmt.Errorf("unknown image provider: %s", req.Provider)
	}
}

type imageProvider interface {
	Generate(ctx context.Context, req *ImageRequest) ([]byte, error)
}

type pollinationsImageProvider struct {
	client *http.Client
}

func (p *pollinationsImageProvider) Generate(ctx context.Context, req *ImageRequest) ([]byte, error) {
	encodedPrompt := url.PathEscape(req.Prompt)
	endpoint := fmt.Sprintf("https://gen.pollinations.ai/image/%s", encodedPrompt)

	params := url.Values{}
	if req.Model != "" {
		params.Set("model", req.Model)
	}
	if req.Width != nil {
		params.Set("width", fmt.Sprintf("%d", *req.Width))
	}
	if req.Height != nil {
		params.Set("height", fmt.Sprintf("%d", *req.Height))
	}
	if req.Seed != nil {
		params.Set("seed", fmt.Sprintf("%d", *req.Seed))
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
