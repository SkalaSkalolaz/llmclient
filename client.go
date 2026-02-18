package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	defaultTimeout         = 120 * time.Second
	defaultOllamaURL       = "http://localhost:11434/v1/chat/completions"
	defaultPollinationsURL = "https://gen.pollinations.ai/v1/chat/completions"
	defaultOpenRouterURL   = "https://openrouter.ai/api/v1/chat/completions"
)

var defaultHTTPClient = &http.Client{Timeout: defaultTimeout}

type Client struct {
	httpClient *http.Client
}

func NewClient(opts ...ClientOption) *Client {
	c := &Client{httpClient: defaultHTTPClient}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type ClientOption func(*Client)

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

type Message struct {
	Role    string
	Content string
}

type Request struct {
	Provider     string
	Model        string
	APIKey       string
	SystemPrompt string
	Prompt       string
	Messages     []Message
	Images       []string
	Endpoint     string
	Temperature  *float64
	MaxTokens    *int
	Seed         *int
}

type Response struct {
	Content string
	Raw     []byte
}

func (c *Client) Send(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	provider, err := c.newProvider(req)
	if err != nil {
		return nil, err
	}

	history := req.Messages
	if len(history) == 0 && req.Prompt != "" {
		history = []Message{{Role: "user", Content: req.Prompt}}
	}

	content, err := provider.Send(ctx, history, req.Images, req.SystemPrompt)
	if err != nil {
		return nil, err
	}

	return &Response{Content: content}, nil
}

func (c *Client) newProvider(req *Request) (provider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "ollama":
		endpoint := req.Endpoint
		if endpoint == "" {
			endpoint = defaultOllamaURL
		}
		return &ollamaProvider{model: req.Model, endpoint: endpoint, client: c.httpClient}, nil
	case "pollinations":
		return &pollinationsProvider{model: req.Model, key: req.APIKey, client: c.httpClient, seed: req.Seed}, nil
	case "openrouter":
		return &openRouterProvider{model: req.Model, key: req.APIKey, client: c.httpClient}, nil
	default:
		if isURL(name) {
			return &genericProvider{endpoint: name, model: req.Model, key: req.APIKey, client: c.httpClient}, nil
		}
		if isURL(req.Endpoint) {
			return &genericProvider{endpoint: req.Endpoint, model: req.Model, key: req.APIKey, client: c.httpClient}, nil
		}
		return nil, fmt.Errorf("unknown provider: %s", req.Provider)
	}
}

type provider interface {
	Send(ctx context.Context, history []Message, images []string, systemPrompt string) (string, error)
}

type ollamaProvider struct {
	model    string
	endpoint string
	client   *http.Client
}

func (p *ollamaProvider) Send(ctx context.Context, history []Message, images []string, systemPrompt string) (string, error) {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs, "stream": false}
	respBody, err := postJSON(ctx, p.client, p.endpoint, payload, "")
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

type pollinationsProvider struct {
	model  string
	key    string
	client *http.Client
	seed   *int
}

func (p *pollinationsProvider) Send(ctx context.Context, history []Message, images []string, systemPrompt string) (string, error) {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs}
	if p.seed != nil {
		payload["seed"] = *p.seed
	}
	respBody, err := postJSON(ctx, p.client, defaultPollinationsURL, payload, p.key)
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

type openRouterProvider struct {
	model  string
	key    string
	client *http.Client
}

func (p *openRouterProvider) Send(ctx context.Context, history []Message, images []string, systemPrompt string) (string, error) {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs}
	respBody, err := postJSON(ctx, p.client, defaultOpenRouterURL, payload, p.key)
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

type genericProvider struct {
	endpoint string
	model    string
	key      string
	client   *http.Client
}

func (p *genericProvider) Send(ctx context.Context, history []Message, images []string, systemPrompt string) (string, error) {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs}
	respBody, err := postJSON(ctx, p.client, p.endpoint, payload, p.key)
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

func messagesToMaps(history []Message, images []string, systemPrompt string) []map[string]interface{} {
	msgs := make([]map[string]interface{}, 0, len(history)+1)
	if systemPrompt != "" {
		msgs = append(msgs, map[string]interface{}{"role": "system", "content": systemPrompt})
	}
	for i, m := range history {
		if i == len(history)-1 && m.Role == "user" && len(images) > 0 {
			msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": buildMessageContent(m.Content, images)})
		} else {
			msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": m.Content})
		}
	}
	return msgs
}

func buildMessageContent(content string, images []string) interface{} {
	if len(images) == 0 {
		return content
	}
	parts := []map[string]interface{}{{"type": "text", "text": content}}
	for _, img := range images {
		parts = append(parts, map[string]interface{}{"type": "image_url", "image_url": map[string]string{"url": img}})
	}
	return parts
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func postJSON(ctx context.Context, client *http.Client, url string, payload interface{}, key string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if strings.Contains(url, "openrouter") {
		req.Header.Set("HTTP-Referer", "https://github.com/llmclient")
		req.Header.Set("X-Title", "LLMClient")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBytes))
	}
	return respBytes, nil
}

func extractContent(body []byte) (string, error) {
	return extractContentFromPossibleJSON(string(body))
}

func extractContentFromPossibleJSON(s string) (string, error) {
	s = strings.TrimSpace(s)
	type GenericResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Content string `json:"content"`
			Text    string `json:"text"`
		} `json:"choices"`
		Content string `json:"content"`
		Text    string `json:"text"`
		Output  string `json:"output"`
		Error   string `json:"error"`
	}
	var r GenericResp
	if err := json.Unmarshal([]byte(s), &r); err == nil {
		if r.Error != "" {
			return "", errors.New(r.Error)
		}
		if len(r.Choices) > 0 {
			if r.Choices[0].Message.Content != "" {
				return r.Choices[0].Message.Content, nil
			}
			if r.Choices[0].Content != "" {
				return r.Choices[0].Content, nil
			}
			if r.Choices[0].Text != "" {
				return r.Choices[0].Text, nil
			}
		}
		if r.Content != "" {
			return r.Content, nil
		}
		if r.Text != "" {
			return r.Text, nil
		}
		if r.Output != "" {
			return r.Output, nil
		}
	}
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		if content, err := extractContentFromPossibleJSON(m[1]); err == nil {
			return content, nil
		}
		return m[1], nil
	}
	if len(s) > 0 && !strings.HasPrefix(s, "{") {
		return s, nil
	}
	return "", errors.New("failed to extract content")
}
