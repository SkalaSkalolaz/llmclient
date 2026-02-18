# llmclient
A minimal, dependency-free Go client for multiple LLM providers with a unified API.

## Features

- **Unified API** - Single function call for all providers
- **Multiple Providers** - Ollama, Pollinations, OpenRouter, and custom endpoints
- **Vision Support** - Send images with your prompts
- **Conversation History** - Multi-turn chat support
- **Zero Dependencies** - Uses only Go standard library
- **Context Support** - Proper cancellation and timeout handling

## Installation

```bash
go get github.com/SkalaSkalolaz/llmclient
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/SkalaSkalolaz/llmclient"
)

func main() {
    // Simple call
    response, err := llmclient.Send(
        "openrouter",                    // provider
        "anthropic/claude-3-haiku",      // model
        "your-api-key",                  // API key
        "You are a helpful assistant.",  // system prompt
        "What is Go?",                   // user prompt
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(response)
}
```

## Providers

### Ollama (Local)

```go
response, err := llmclient.Send("ollama", "llama3", "", "You are helpful.", "Hello!")
```

### Pollinations

```go
response, err := llmclient.Send("pollinations", "openai", "", "system", "prompt",
    llmclient.WithSeed(42))
```

### OpenRouter

```go
response, err := llmclient.Send("openrouter", "anthropic/claude-3-opus", "api-key", "system", "prompt")
```

### Custom Endpoint

```go
// Pass URL as provider or use WithEndpoint
response, err := llmclient.Send("https://api.example.com/v1/chat/completions", "model", "key", "system", "prompt")

// Or with explicit provider name
response, err := llmclient.Send("custom", "model", "key", "system", "prompt",
    llmclient.WithEndpoint("https://api.example.com/v1/chat/completions"))
```

## Usage Examples

### With Options

```go
response, err := llmclient.Send("openrouter", "gpt-4", "key", "system", "prompt",
    llmclient.WithSeed(42),
    llmclient.WithTemperature(0.7),
    llmclient.WithMaxTokens(1000),
)
```

### With Context and Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := llmclient.SendWithContext(ctx, "ollama", "llama3", "", "system", "prompt")
```

### Vision (Images)

```go
// Images in OpenAI format (base64 data URI or URL)
images := []string{
    "data:image/png;base64,iVBORw0KGgo...",
    "https://example.com/image.png",
}

response, err := llmclient.SendWithImages("openrouter", "gpt-4-vision-preview", "key", 
    "system", "What's in this image?", images)
```

### Conversation History

```go
messages := []llmclient.Message{
    {Role: "user", Content: "Hello!"},
    {Role: "assistant", Content: "Hi! How can I help you?"},
    {Role: "user", Content: "Tell me about Go."},
}

response, err := llmclient.SendMessages("openrouter", "gpt-4", "key", "You are helpful.", messages)
```

### Using Client for Multiple Requests

```go
client := llmclient.NewClient(
    llmclient.WithTimeout(60 * time.Second),
    llmclient.WithHTTPClient(customHTTPClient),
)

resp, err := client.Send(ctx, &llmclient.Request{
    Provider:     "openrouter",
    Model:        "gpt-4",
    APIKey:       "key",
    SystemPrompt: "You are helpful.",
    Prompt:       "Hello!",
})
fmt.Println(resp.Content)
```

## API Reference

### Simple Functions

| Function | Description |
|----------|-------------|
| `Send(provider, model, apiKey, systemPrompt, prompt string, opts ...SendOption) (string, error)` | Single message |
| `SendWithContext(...)` | With context |
| `SendWithImages(..., images []string)` | With images |
| `SendMessages(..., messages []Message)` | With conversation history |

### Options

| Option | Description |
|--------|-------------|
| `WithImages(images []string)` | Attach images |
| `WithEndpoint(url string)` | Custom API endpoint |
| `WithTemperature(temp float64)` | Sampling temperature |
| `WithMaxTokens(max int)` | Max tokens in response |
| `WithSeed(seed int)` | Deterministic sampling |

### Client Options

| Option | Description |
|--------|-------------|
| `WithTimeout(d time.Duration)` | HTTP timeout |
| `WithHTTPClient(c *http.Client)` | Custom HTTP client |

## License

Massachusetts Institute of Technology
