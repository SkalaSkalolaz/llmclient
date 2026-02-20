# llmclient

Minimal stdlib-only Go client with a unified API for OpenAI-compatible LLM endpoints.

- Русская версия: [README_RU.md](./README_RU.md)

## Features

- **Chat (non-stream)**: one API for Ollama, OpenRouter, Pollinations, and any OpenAI-compatible URL
- **Streaming (SSE)**: token/chunk streaming via callback
- **Conversation history**: send `[]Message`
- **Vision**: images as URL or `data:image/...;base64,...`, plus `ContentPart` API
- **Image generation (Pollinations)**: `gen.pollinations.ai/image/{prompt}` (+ width/height/seed)
- **Audio generation (Pollinations)**: `gen.pollinations.ai/audio/{prompt}` (+ model)
- **Audio transcription (Pollinations)**: multipart upload to `gen.pollinations.ai/v1/audio/transcriptions`
- **Models**: list text/audio models (Pollinations)
- **Account** (Pollinations): profile, balance, usage (JSON/CSV)
- **Context**: cancellation/timeouts via `context.Context`
- **Zero dependencies**: standard library only

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
_ = os.WriteFile("output.png", imageData, 0644)
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

Reuse HTTP connections across multiple requests (chat/stream/image/audio/transcription):

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

## Audio

### Generate audio (Pollinations)

```go
wav, err := llmclient.GenerateAudio("pollinations", "", "Hello from Go!",
    llmclient.WithAudioModel("elevenlabs"),
)
if err != nil {
    log.Fatal(err)
}
_ = os.WriteFile("out.wav", wav, 0644)
```

### Transcribe audio (Pollinations)

```go
b, _ := os.ReadFile("speech.wav")

c := llmclient.NewClient()
resp, err := c.TranscribeAudio(context.Background(), &llmclient.TranscriptionRequest{
    Provider: "pollinations",
    APIKey:   "",
    Model:    "whisper-1",
    FileName: "speech.wav",
    FileData: b,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Text)
```

## Models

List available models (Pollinations):

```go
models, err := llmclient.ListTextModels("pollinations", "")
if err != nil {
    log.Fatal(err)
}
fmt.Println("models:", len(models))

audioModels, err := llmclient.ListAudioModels("pollinations", "")
if err != nil {
    log.Fatal(err)
}
fmt.Println("audio models:", len(audioModels))
```

Helpers:

```go
free := llmclient.FilterFreeModels(models)
tts := llmclient.FilterTextToSpeechModels(audioModels)
```

## Account (Pollinations)

```go
bal, err := llmclient.GetBalance("pollinations", "your-api-key")
if err != nil {
    log.Fatal(err)
}
fmt.Println(bal.Credits, bal.Currency)

profile, err := llmclient.GetProfile("pollinations", "your-api-key")
if err != nil {
    log.Fatal(err)
}
fmt.Println(profile.Email, profile.Plan)

usage, err := llmclient.GetUsage("pollinations", "your-api-key", llmclient.UsageFormatJSON)
if err != nil {
    log.Fatal(err)
}
fmt.Println(usage.Totals.TotalTokens, usage.Totals.TotalCost)
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

### Audio Generation

| Function | Description |
|----------|-------------|
| `GenerateAudio(provider, apiKey, prompt, opts...)` | Generate audio |
| `GenerateAudioWithContext(ctx, ...)` | With context |

### Audio Transcription

| Method | Description |
|--------|-------------|
| `(*Client).TranscribeAudio(ctx, req)` | Transcribe audio file (Pollinations) |

### Models

| Function | Description |
|----------|-------------|
| `ListTextModels(provider, apiKey)` | List text/chat models |
| `ListAudioModels(provider, apiKey)` | List audio models |

### Account (Pollinations)

| Function | Description |
|----------|-------------|
| `GetProfile(provider, apiKey)` | Get account profile |
| `GetBalance(provider, apiKey)` | Get account balance/credits |
| `GetUsage(provider, apiKey, format)` | Get usage (JSON/CSV) |

### Options

Note: these options are set on `Request`, but **not all providers forward them to the upstream payload yet**.

| Option | Description |
|--------|-------------|
| `WithImages(images)` | Attach images to request |
| `WithEndpoint(url)` | Custom API endpoint |
| `WithTemperature(temp)` | Sampling temperature (пока не пробрасывается в payload) |
| `WithMaxTokens(max)` | Max tokens in response (пока не пробрасывается в payload) |
| `WithSeed(seed)` | Seed (используется Pollinations) |

### Image Options

| Option | Description |
|--------|-------------|
| `WithImageWidth(width)` | Image width in pixels |
| `WithImageHeight(height)` | Image height in pixels |
| `WithImageSeed(seed)` | Seed for reproducibility |

### Audio Options

| Option | Description |
|--------|-------------|
| `WithAudioModel(model)` | Audio model name (Pollinations query param) |

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
