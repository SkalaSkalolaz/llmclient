package llmclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type StreamChunk struct {
	Content string
	Done    bool
}

type StreamCallback func(chunk StreamChunk) error

type StreamResponse struct {
	Content string
}

func (c *Client) SendStream(ctx context.Context, req *Request, callback StreamCallback) (*StreamResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if callback == nil {
		return nil, errors.New("callback is nil")
	}

	provider, err := c.newStreamProvider(req)
	if err != nil {
		return nil, err
	}

	history := req.Messages
	if len(history) == 0 && req.Prompt != "" {
		history = []Message{{Role: "user", Content: req.Prompt}}
	}

	var fullContent strings.Builder
	err = provider.SendStream(ctx, history, req.Images, req.SystemPrompt, func(chunk StreamChunk) error {
		if !chunk.Done {
			fullContent.WriteString(chunk.Content)
		}
		return callback(chunk)
	})
	if err != nil {
		return nil, err
	}

	return &StreamResponse{Content: fullContent.String()}, nil
}

func (c *Client) newStreamProvider(req *Request) (streamingProvider, error) {
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

type streamingProvider interface {
	SendStream(ctx context.Context, history []Message, images []string, systemPrompt string, callback StreamCallback) error
}

func (p *ollamaProvider) SendStream(ctx context.Context, history []Message, images []string, systemPrompt string, callback StreamCallback) error {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs, "stream": true}
	return postJSONStream(ctx, p.client, p.endpoint, payload, "", callback)
}

func (p *pollinationsProvider) SendStream(ctx context.Context, history []Message, images []string, systemPrompt string, callback StreamCallback) error {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs, "stream": true}
	if p.seed != nil {
		payload["seed"] = *p.seed
	}

	endpoint := pollinationsPaidURL
	if p.key == "" {
		endpoint = pollinationsFreeURL
	}

	return postJSONStream(ctx, p.client, endpoint, payload, p.key, callback)
}

func (p *openRouterProvider) SendStream(ctx context.Context, history []Message, images []string, systemPrompt string, callback StreamCallback) error {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs, "stream": true}
	return postJSONStream(ctx, p.client, defaultOpenRouterURL, payload, p.key, callback)
}

func (p *genericProvider) SendStream(ctx context.Context, history []Message, images []string, systemPrompt string, callback StreamCallback) error {
	msgs := messagesToMaps(history, images, systemPrompt)
	payload := map[string]interface{}{"model": p.model, "messages": msgs, "stream": true}
	return postJSONStream(ctx, p.client, p.endpoint, payload, p.key, callback)
}

func postJSONStream(ctx context.Context, client *http.Client, url string, payload interface{}, key string, callback StreamCallback) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if strings.Contains(url, "openrouter") {
		req.Header.Set("HTTP-Referer", "https://github.com/llmclient")
		req.Header.Set("X-Title", "LLMClient")
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBytes))
	}

	return parseSSEStream(resp.Body, callback)
}

func parseSSEStream(reader io.Reader, callback StreamCallback) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			if err := callback(StreamChunk{Done: true}); err != nil {
				return err
			}
			break
		}

		content, err := extractStreamContent(data)
		if err != nil {
			continue
		}

		if content != "" {
			if err := callback(StreamChunk{Content: content}); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

func extractStreamContent(data string) (string, error) {
	type StreamResp struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	var r StreamResp
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return "", err
	}

	if len(r.Choices) > 0 {
		return r.Choices[0].Delta.Content, nil
	}

	return "", nil
}
