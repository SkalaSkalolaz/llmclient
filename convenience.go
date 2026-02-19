package llmclient

import (
	"context"
	"net/http"
	"time"
)

func Send(provider, model, apiKey, systemPrompt, prompt string, opts ...SendOption) (string, error) {
	return SendWithContext(context.Background(), provider, model, apiKey, systemPrompt, prompt, opts...)
}

func SendWithContext(ctx context.Context, provider, model, apiKey, systemPrompt, prompt string, opts ...SendOption) (string, error) {
	req := &Request{
		Provider:     provider,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		Prompt:       prompt,
	}
	for _, opt := range opts {
		opt(req)
	}
	client := NewClient()
	resp, err := client.Send(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func SendMessages(provider, model, apiKey, systemPrompt string, messages []Message, opts ...SendOption) (string, error) {
	return SendMessagesWithContext(context.Background(), provider, model, apiKey, systemPrompt, messages, opts...)
}

func SendMessagesWithContext(ctx context.Context, provider, model, apiKey, systemPrompt string, messages []Message, opts ...SendOption) (string, error) {
	req := &Request{
		Provider:     provider,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		Messages:     messages,
	}
	for _, opt := range opts {
		opt(req)
	}
	client := NewClient()
	resp, err := client.Send(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func SendWithImages(provider, model, apiKey, systemPrompt, prompt string, images []string, opts ...SendOption) (string, error) {
	return SendWithImagesWithContext(context.Background(), provider, model, apiKey, systemPrompt, prompt, images, opts...)
}

func SendWithImagesWithContext(ctx context.Context, provider, model, apiKey, systemPrompt, prompt string, images []string, opts ...SendOption) (string, error) {
	req := &Request{
		Provider:     provider,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		Prompt:       prompt,
		Images:       images,
	}
	for _, opt := range opts {
		opt(req)
	}
	client := NewClient()
	resp, err := client.Send(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

type SendOption func(*Request)

func WithImages(images []string) SendOption {
	return func(r *Request) { r.Images = images }
}

func WithEndpoint(endpoint string) SendOption {
	return func(r *Request) { r.Endpoint = endpoint }
}

func WithTemperature(temp float64) SendOption {
	return func(r *Request) { r.Temperature = &temp }
}

func WithMaxTokens(max int) SendOption {
	return func(r *Request) { r.MaxTokens = &max }
}

func WithSeed(seed int) SendOption {
	return func(r *Request) { r.Seed = &seed }
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient = &http.Client{Timeout: timeout}
	}
}

func GenerateImage(provider, model, apiKey, prompt string, opts ...ImageOption) ([]byte, error) {
	return GenerateImageWithContext(context.Background(), provider, model, apiKey, prompt, opts...)
}

func GenerateImageWithContext(ctx context.Context, provider, model, apiKey, prompt string, opts ...ImageOption) ([]byte, error) {
	req := &ImageRequest{
		Provider: provider,
		Model:    model,
		APIKey:   apiKey,
		Prompt:   prompt,
	}
	for _, opt := range opts {
		opt(req)
	}
	client := NewClient()
	resp, err := client.GenerateImage(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

type ImageOption func(*ImageRequest)

func WithImageWidth(width int) ImageOption {
	return func(r *ImageRequest) { r.Width = &width }
}

func WithImageHeight(height int) ImageOption {
	return func(r *ImageRequest) { r.Height = &height }
}

func WithImageSeed(seed int) ImageOption {
	return func(r *ImageRequest) { r.Seed = &seed }
}

func NewUserMessage(text string) Message {
	return Message{Role: "user", Content: text}
}

func NewAssistantMessage(text string) Message {
	return Message{Role: "assistant", Content: text}
}

func NewSystemMessage(text string) Message {
	return Message{Role: "system", Content: text}
}

func NewUserMessageWithImages(text string, imageUrls []string) Message {
	parts := []ContentPart{NewTextPart(text)}
	for _, url := range imageUrls {
		parts = append(parts, NewImageURLPart(url))
	}
	return Message{Role: "user", Content: text, ContentParts: parts}
}

func NewUserMessageWithContentParts(parts []ContentPart) Message {
	var textContent string
	for _, p := range parts {
		if p.Type == "text" {
			textContent = p.Text
			break
		}
	}
	return Message{Role: "user", ContentParts: parts, Content: textContent}
}

func GenerateAudio(provider, apiKey, prompt string, opts ...AudioOption) ([]byte, error) {
	return GenerateAudioWithContext(context.Background(), provider, apiKey, prompt, opts...)
}

func GenerateAudioWithContext(ctx context.Context, provider, apiKey, prompt string, opts ...AudioOption) ([]byte, error) {
	req := &AudioRequest{
		Provider: provider,
		APIKey:   apiKey,
		Prompt:   prompt,
	}
	for _, opt := range opts {
		opt(req)
	}
	client := NewClient()
	resp, err := client.GenerateAudio(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

type AudioOption func(*AudioRequest)

func WithAudioModel(model string) AudioOption {
	return func(r *AudioRequest) { r.Model = model }
}

func SendStream(provider, model, apiKey, systemPrompt, prompt string, callback StreamCallback, opts ...SendOption) (string, error) {
	return SendStreamWithContext(context.Background(), provider, model, apiKey, systemPrompt, prompt, callback, opts...)
}

func SendStreamWithContext(ctx context.Context, provider, model, apiKey, systemPrompt, prompt string, callback StreamCallback, opts ...SendOption) (string, error) {
	req := &Request{
		Provider:     provider,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		Prompt:       prompt,
	}
	for _, opt := range opts {
		opt(req)
	}
	client := NewClient()
	resp, err := client.SendStream(ctx, req, callback)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func SendMessagesStream(provider, model, apiKey, systemPrompt string, messages []Message, callback StreamCallback, opts ...SendOption) (string, error) {
	return SendMessagesStreamWithContext(context.Background(), provider, model, apiKey, systemPrompt, messages, callback, opts...)
}

func SendMessagesStreamWithContext(ctx context.Context, provider, model, apiKey, systemPrompt string, messages []Message, callback StreamCallback, opts ...SendOption) (string, error) {
	req := &Request{
		Provider:     provider,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		Messages:     messages,
	}
	for _, opt := range opts {
		opt(req)
	}
	client := NewClient()
	resp, err := client.SendStream(ctx, req, callback)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
