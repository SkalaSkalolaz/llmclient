package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

type TranscriptionRequest struct {
	Provider       string
	Model          string
	APIKey         string
	FileName       string
	FileData       []byte
	Language       string
	Prompt         string
	ResponseFormat string
	Temperature    *float64
}

type TranscriptionResponse struct {
	Text string
	Raw  []byte
}

func (c *Client) TranscribeAudio(ctx context.Context, req *TranscriptionRequest) (*TranscriptionResponse, error) {
	if req == nil {
		return nil, errors.New("transcription request is nil")
	}

	provider, err := c.newTranscriptionProvider(req)
	if err != nil {
		return nil, err
	}

	text, raw, err := provider.Transcribe(ctx, req)
	if err != nil {
		return nil, err
	}

	return &TranscriptionResponse{Text: text, Raw: raw}, nil
}

func (c *Client) newTranscriptionProvider(req *TranscriptionRequest) (transcriptionProvider, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))

	switch name {
	case "pollinations":
		return &pollinationsTranscriptionProvider{client: c.httpClient}, nil
	default:
		return nil, fmt.Errorf("unknown transcription provider: %s", req.Provider)
	}
}

type transcriptionProvider interface {
	Transcribe(ctx context.Context, req *TranscriptionRequest) (string, []byte, error)
}

type pollinationsTranscriptionProvider struct {
	client *http.Client
}

func (p *pollinationsTranscriptionProvider) Transcribe(ctx context.Context, req *TranscriptionRequest) (string, []byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("file", filepath.Base(req.FileName))
	if err != nil {
		return "", nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := fileWriter.Write(req.FileData); err != nil {
		return "", nil, fmt.Errorf("write file data: %w", err)
	}

	if req.Model != "" {
		_ = writer.WriteField("model", req.Model)
	}
	if req.Language != "" {
		_ = writer.WriteField("language", req.Language)
	}
	if req.Prompt != "" {
		_ = writer.WriteField("prompt", req.Prompt)
	}
	if req.ResponseFormat != "" {
		_ = writer.WriteField("response_format", req.ResponseFormat)
	}
	if req.Temperature != nil {
		_ = writer.WriteField("temperature", fmt.Sprintf("%.2f", *req.Temperature))
	}

	if err := writer.Close(); err != nil {
		return "", nil, fmt.Errorf("close multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://gen.pollinations.ai/v1/audio/transcriptions", &body)
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respData))
	}

	text := extractTranscriptionText(respData)
	return text, respData, nil
}

func extractTranscriptionText(data []byte) string {
	type TranscriptionResult struct {
		Text string `json:"text"`
	}
	var result TranscriptionResult
	if err := json.Unmarshal(data, &result); err == nil && result.Text != "" {
		return result.Text
	}
	return string(data)
}
