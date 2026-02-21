# CodeBridge

> AI-powered code translation service with streaming support

CodeBridge is a production-ready Go service that translates code between programming languages using AI providers (OpenAI GPT or Google Gemini). It features a pluggable provider system, Server-Sent Events (SSE) for real-time streaming, and a clean architecture designed for scalability.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Features

-  **Multi-Provider Support** - Switch between multiple LLM providers with a factory pattern
-  **Real-time Streaming** - SSE-based streaming for live translation updates
- ️ **Clean Architecture** - Modular design with dependency injection
-  **Provider Factory Pattern** - Easily add new AI providers

## Quick Start

### Prerequisites

- Go 1.24 or higher
- PostgreSQL 14+
- OpenAI API key or Google Gemini API key

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/yourusername/code-bridge.git
cd code-bridge
```

2. **Copy environment file**
```bash
cp .env.example .env
```

3. **Configure environment variables**
```bash
# Edit .env with your settings
# Required: Database credentials and at least one AI provider API key
nano .env
```

4. **Install dependencies**
```bash
make deps
```

5. **Run the server**
```bash
make dev
# or
make build && ./code-bridge
```

The server starts at `http://localhost:6777`

### Basic Usage

**1. Initiate a translation**
```bash
curl -X POST http://localhost:6777/translate \
  -H "Content-Type: application/json" \
  -d '{
    "code": "def hello():\n    print(\"Hello\")",
    "source_language": "python",
    "target_language": "javascript"
  }'
```

**Response:**
```json
{
  "id": "job-1704412800000000000"
}
```

**2. Stream the translation results**
```bash
curl http://localhost:6777/translate/stream/job-1704412800000000000
```

**SSE Stream Output:**
```
: connected
data: {"type":"explanation","content":"<content>","delta":true}
...
data: {"type":"notes","content":"<content>"}
...
data: {"type":"code","content":"<content>","delta":true}
...
data: [DONE]
```

## Architecture

### Project Structure

```
code-bridge/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   └── gin_server.go          # HTTP handlers and routes
│   ├── code_translator/
│   │   └── code_translator_service.go  # Translation business logic
│   ├── services/
│   │   └── services.go            # Service container
│   ├── sse/
│   │   └── hub.go                 # Server-Sent Events hub
│   ├── translator_provider/
│   │   ├── provider.go            # Provider interface
│   │   └── factory.go             # Provider factory
│   └── third_party/
│       ├── openai/
│       │   └── client.go          # OpenAI integration
│       └── gemini/
│           └── client.go          # Gemini integration
├── pkg/
│   ├── database/
│   │   └── postgres.go            # Database connection
│   └── types/
│       ├── config.go              # Configuration types
│       └── request.go             # Request/response types
├── web/
│   ├── index.html                 # Demo web interface
│   └── static/                    # Static assets
├── .env.example                    # Environment template
├── Makefile                        # Build automation
└── go.mod                          # Go module definition
```

### Component Overview

#### Provider Factory System

The provider factory allows seamless switching between AI providers:

```go
// Define provider type
type TranslatorProvider interface {
    StreamCompletion(ctx context.Context, prompt string, onChunk func(string) error) error
}

// Create provider
factory := translator_provider.NewFactory(config)
provider, _ := factory.CreateProvider(translator_provider.ProviderOpenAI)
// or
provider, _ := factory.CreateProvider(translator_provider.ProviderGemini)
```

#### SSE Hub

Real-time streaming using Server-Sent Events:

```go
// Create stream
hub.Create("job-id")

// Send chunks
hub.Send("job-id", "translation chunk")

// Subscribe client
client := hub.AddClient("job-id")
```

#### Service Layer

Business logic isolated from HTTP handlers:

```go
translatorService.TranslateCode(
    ctx, 
    code, 
    sourceLang, 
    targetLang, 
    func(chunk string) error {
        return hub.Send(jobID, chunk)
    },
)
```

## API Reference

### Endpoints

#### `GET /health`
Health check endpoint

**Response:**
```json
{
  "status": "healthy",
  "service": "codebridge-api"
}
```

#### `POST /translate`
Initiate code translation

**Request Body:**
```json
{
  "code": "string (required)",
  "source_language": "string (optional)",
  "target_language": "string (required)"
}
```

**Response:**
```json
{
  "id": "job-1704412800000000000"
}
```

#### `GET /translate/stream/:id`
Stream translation results via SSE

**Response:** Server-Sent Events stream
```
data: <chunk>
...
data: [DONE]
```

#### `GET /web`
Demo web interface

## Configuration

### Environment Variables

Create a `.env` file from `.env.example`:

### Provider Selection

Change the provider in `cmd/server/main.go`:

```go
// Use OpenAI
provider, err := providerFactory.CreateProvider(translator_provider.ProviderOpenAI)

// Use Gemini
provider, err := providerFactory.CreateProvider(translator_provider.ProviderGemini)
```

## Development

### Available Commands

```bash
make build      # Build the application
make run        # Build and run
make dev        # Run without building (hot reload with air/reflex)
make clean      # Clean build artifacts
make deps       # Download dependencies
make tidy       # Tidy and verify dependencies
```

### Adding a New Provider

1. **Create client implementation**
```go
// internal/third_party/newprovider/client.go
type Client struct {
    apiKey string
}

func (c *Client) StreamCompletion(ctx context.Context, prompt string, onChunk func(string) error) error {
    // Implement streaming logic
}
```

2. **Add configuration**
```go
// pkg/types/config.go
type NewProviderConfig struct {
    APIKey string
}
```

3. **Register in factory**
```go
// internal/translator_provider/factory.go
case ProviderNewProvider:
    return newprovider.NewClient(config.NewProvider)
```

## Key Dependencies

- **[Gin](https://github.com/gin-gonic/gin)** - HTTP web framework
- **[Zap](https://github.com/uber-go/zap)** - Structured logging
- **[Viper](https://github.com/spf13/viper)** - Configuration management
- **[Bun](https://github.com/uptrace/bun)** - PostgreSQL ORM
- **[OpenAI Go SDK](https://github.com/openai/openai-go)** - OpenAI integration
- **[Google Generative AI](https://pkg.go.dev/google.golang.org/genai)** - Gemini integration

## License

MIT License - see [LICENSE](LICENSE) file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please use the [GitHub Issues](https://github.com/ayomideajewole/code-bridge/issues) page.
