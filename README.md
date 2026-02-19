# llmclient

Minimal Go client for multiple LLM providers with unified API. Zero dependencies.

## Features

- **Unified API** — single interface for all providers
- **Streaming** — SSE streaming with callbacks
- **Multiple Providers** — Ollama, Pollinations, OpenRouter, custom endpoints
- **Vision** — images via URLs or base64
- **Image Generation** — generate images via Pollinations
- **Conversation History** — multi-turn chat support
- **Context Support** — proper cancellation and timeouts
- **Zero Dependencies** — standard library only

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
    response, err := llmclient.Send(
        "openrouter",
        "anthropic/claude-3-haiku",
        "your-api-key",
        "You are a helpful assistant.",
        "What is Go?",
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

Custom endpoint:
```go
response, err := llmclient.Send("ollama", "llama3", "", "system", "prompt",
    llmclient.WithEndpoint("http://your-server:11434/v1/chat/completions"))
```

### Pollinations

Two modes:
- **Free** (no API key) → `text.pollinations.ai/openai`
- **Paid** (with API key) → `gen.pollinations.ai/v1/chat/completions`

```go
response, err := llmclient.Send("pollinations", "openai", "", "system", "prompt",
    llmclient.WithSeed(42))
```

### OpenRouter

```go
response, err := llmclient.Send("openrouter", "anthropic/claude-3-opus", "api-key", "system", "prompt")
```

### Custom Endpoint

Any OpenAI-compatible API:
```go
response, err := llmclient.Send("https://api.example.com/v1/chat/completions", "model", "key", "system", "prompt")

// Or with explicit provider
response, err := llmclient.Send("custom", "model", "key", "system", "prompt",
    llmclient.WithEndpoint("https://api.example.com/v1/chat/completions"))
```

## Text Generation

### Simple Call

```go
response, err := llmclient.Send("openrouter", "gpt-4", "key", "system", "prompt")
```

### With Options

```go
response, err := llmclient.Send("openrouter", "gpt-4", "key", "system", "prompt",
    llmclient.WithSeed(42),
    llmclient.WithTemperature(0.7),
    llmclient.WithMaxTokens(1000),
)
```

### With Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := llmclient.SendWithContext(ctx, "ollama", "llama3", "", "system", "prompt")
```

### Conversation History

```go
messages := []llmclient.Message{
    llmclient.NewUserMessage("Hello!"),
    llmclient.NewAssistantMessage("Hi! How can I help you?"),
    llmclient.NewUserMessage("Tell me about Go."),
}

response, err := llmclient.SendMessages("openrouter", "gpt-4", "key", "You are helpful.", messages)
```

## Streaming

```go
fullResponse, err := llmclient.SendStream("pollinations", "openai", "", "You are a poet.",
    "Write a poem about Go",
    func(chunk llmclient.StreamChunk) error {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
        if chunk.Done {
            fmt.Println("\n[Done]")
        }
        return nil
    })
```

With context and history:
```go
messages := []llmclient.Message{llmclient.NewUserMessage("Tell me a story")}

ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

_, err := llmclient.SendMessagesStreamWithContext(ctx, "pollinations", "openai", "",
    "You are a storyteller.", messages,
    func(chunk llmclient.StreamChunk) error {
        fmt.Print(chunk.Content)
        return nil
    })
```

## Vision (Images)

### Simple with URLs

```go
images := []string{
    "data:image/png;base64,iVBORw0KGgo...",
    "https://example.com/image.png",
}

response, err := llmclient.SendWithImages("openrouter", "gpt-4-vision-preview", "key",
    "system", "What's in this image?", images)
```

### Content Parts API

Fine-grained control with `ContentPart`:

```go
parts := []llmclient.ContentPart{
    llmclient.NewTextPart("What's in these images?"),
    llmclient.NewImageURLPart("https://example.com/image1.png"),
    llmclient.NewImageURLPartWithDetail("https://example.com/image2.png", "high"),
    llmclient.NewImageBase64Part("image/jpeg", base64Data),
}

msg := llmclient.NewUserMessageWithContentParts(parts)

response, err := llmclient.SendMessages("openrouter", "gpt-4-vision-preview", "key", "", []llmclient.Message{msg})
```

Helper for user messages with images:
```go
msg := llmclient.NewUserMessageWithImages("Describe this", []string{"https://example.com/img.png"})
```

## Image Generation

Generate images via Pollinations:

```go
imageData, err := llmclient.GenerateImage("pollinations", "flux", "", "A sunset over mountains")
if err != nil {
    log.Fatal(err)
}
err = os.WriteFile("output.png", imageData, 0644)
```

With options:
```go
imageData, err := llmclient.GenerateImage("pollinations", "flux", "", "A sunset",
    llmclient.WithImageWidth(1024),
    llmclient.WithImageHeight(768),
    llmclient.WithImageSeed(42),
)
```

With context:
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

imageData, err := llmclient.GenerateImageWithContext(ctx, "pollinations", "flux", "", "prompt")
```

## Client Instance

Reuse HTTP connections across multiple requests:

```go
client := llmclient.NewClient(
    llmclient.WithTimeout(60 * time.Second),
)

resp, err := client.Send(ctx, &llmclient.Request{
    Provider:     "openrouter",
    Model:        "gpt-4",
    APIKey:       "key",
    SystemPrompt: "You are helpful.",
    Prompt:       "Hello!",
})
fmt.Println(resp.Content)

imgResp, err := client.GenerateImage(ctx, &llmclient.ImageRequest{
    Provider: "pollinations",
    Prompt:   "A mountain landscape",
})
os.WriteFile("output.png", imgResp.Data, 0644)

streamResp, err := client.SendStream(ctx, &llmclient.Request{
    Provider: "pollinations",
    Model:    "openai",
    Prompt:   "Tell me a joke",
}, func(chunk llmclient.StreamChunk) error {
    fmt.Print(chunk.Content)
    return nil
})
```

Custom HTTP client:
```go
customHTTP := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns: 10,
    },
}

client := llmclient.NewClient(llmclient.WithHTTPClient(customHTTP))
```

## API Reference

### Simple Functions

| Function | Description |
|----------|-------------|
| `Send(provider, model, apiKey, systemPrompt, prompt, opts...)` | Single message |
| `SendWithContext(ctx, ...)` | With context |
| `SendWithImages(..., images)` | With images |
| `SendWithImagesWithContext(ctx, ...)` | With context and images |
| `SendMessages(..., messages)` | With conversation history |
| `SendMessagesWithContext(ctx, ...)` | With context and history |

### Streaming Functions

| Function | Description |
|----------|-------------|
| `SendStream(..., callback)` | Stream response |
| `SendStreamWithContext(ctx, ...)` | Stream with context |
| `SendMessagesStream(..., messages, callback)` | Stream with history |
| `SendMessagesStreamWithContext(ctx, ...)` | Stream with context and history |

### Image Generation

| Function | Description |
|----------|-------------|
| `GenerateImage(provider, model, apiKey, prompt, opts...)` | Generate image |
| `GenerateImageWithContext(ctx, ...)` | With context |

### Options

| Option | Description |
|--------|-------------|
| `WithImages(images)` | Attach images to request |
| `WithEndpoint(url)` | Custom API endpoint |
| `WithTemperature(temp)` | Sampling temperature |
| `WithMaxTokens(max)` | Max tokens in response |
| `WithSeed(seed)` | Deterministic sampling |

### Image Options

| Option | Description |
|--------|-------------|
| `WithImageWidth(width)` | Image width in pixels |
| `WithImageHeight(height)` | Image height in pixels |
| `WithImageSeed(seed)` | Seed for reproducibility |

### Client Options

| Option | Description |
|--------|-------------|
| `WithTimeout(d)` | HTTP timeout |
| `WithHTTPClient(c)` | Custom HTTP client |

### Content Part Constructors

| Function | Description |
|----------|-------------|
| `NewTextPart(text)` | Text content part |
| `NewImageURLPart(url)` | Image from URL |
| `NewImageURLPartWithDetail(url, detail)` | Image with detail level |
| `NewImageBase64Part(mediaType, data)` | Image from base64 |

### Message Constructors

| Function | Description |
|----------|-------------|
| `NewUserMessage(text)` | User message |
| `NewAssistantMessage(text)` | Assistant message |
| `NewSystemMessage(text)` | System message |
| `NewUserMessageWithImages(text, urls)` | User message with images |
| `NewUserMessageWithContentParts(parts)` | User message with content parts |

## License

MIT
