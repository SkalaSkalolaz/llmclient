# llmclient

Минималистичный Go-клиент (stdlib only) с единым API для OpenAI-совместимых LLM-эндпоинтов.

## Возможности

- **Чат (non-stream)**: единая функция для Ollama, OpenRouter, Pollinations и любого OpenAI-compatible URL
- **Streaming (SSE)**: получение ответа чанками через callback
- **История диалога**: отправка массива сообщений (`[]Message`)
- **Vision**: передача изображений как URL или `data:image/...;base64,...`, а также через `ContentPart`
- **Генерация изображений (Pollinations)**: `gen.pollinations.ai/image/{prompt}` (+ width/height/seed)
- **Генерация аудио (Pollinations)**: `gen.pollinations.ai/audio/{prompt}` (+ model)
- **Транскрибация аудио (Pollinations)**: multipart upload в `gen.pollinations.ai/v1/audio/transcriptions`
- **Модели**: получение списка текстовых/аудио моделей (Pollinations)
- **Аккаунт (Pollinations)**: профиль, баланс, usage (JSON/CSV)
- **Context**: корректная отмена/таймауты через `context.Context`
- **0 зависимостей**: только стандартная библиотека

## Установка

```bash
go get github.com/SkalaSkalolaz/llmclient
```

## Быстрый старт

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

## Провайдеры

### Ollama (локально)

```go
response, err := llmclient.Send("ollama", "llama3", "", "You are helpful.", "Hello!")
```

Кастомный endpoint:
```go
response, err := llmclient.Send("ollama", "llama3", "", "system", "prompt",
    llmclient.WithEndpoint("http://your-server:11434/v1/chat/completions"))
```

### Pollinations

Режимы:
- **Free** (без API key) → `text.pollinations.ai/openai`
- **Paid** (с API key) → `gen.pollinations.ai/v1/chat/completions`

```go
response, err := llmclient.Send("pollinations", "openai", "", "system", "prompt",
    llmclient.WithSeed(42))
```

### OpenRouter

```go
response, err := llmclient.Send("openrouter", "anthropic/claude-3-opus", "api-key", "system", "prompt")
```

### Custom Endpoint

Любой OpenAI-compatible API:
```go
response, err := llmclient.Send("https://api.example.com/v1/chat/completions", "model", "key", "system", "prompt")

// Или с явным provider
response, err := llmclient.Send("custom", "model", "key", "system", "prompt",
    llmclient.WithEndpoint("https://api.example.com/v1/chat/completions"))
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

## Vision

```go
images := []string{
    "data:image/png;base64,iVBORw0KGgo...",
    "https://example.com/image.png",
}

response, err := llmclient.SendWithImages("openrouter", "gpt-4-vision-preview", "key",
    "system", "What's in this image?", images)
```

## Генерация изображений

```go
imageData, err := llmclient.GenerateImage("pollinations", "flux", "", "A sunset over mountains")
if err != nil {
    log.Fatal(err)
}
_ = os.WriteFile("output.png", imageData, 0644)
```

## Audio

### Генерация аудио (Pollinations)

```go
wav, err := llmclient.GenerateAudio("pollinations", "", "Hello from Go!",
    llmclient.WithAudioModel("elevenlabs"),
)
if err != nil {
    log.Fatal(err)
}
_ = os.WriteFile("out.wav", wav, 0644)
```

### Транскрибация аудио (Pollinations)

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

## Модели (Pollinations)

Получить список доступных моделей:

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

Полезные фильтры:

```go
free := llmclient.FilterFreeModels(models)
tts := llmclient.FilterTextToSpeechModels(audioModels)
```

## Аккаунт (Pollinations)

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

## Справка по API

См. `README.md` (англ) для полного справочника функций/опций.

## License

MIT
