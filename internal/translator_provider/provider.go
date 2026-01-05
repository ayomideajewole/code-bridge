package translator_provider

import "context"

// TranslatorProvider defines the interface that all translation providers must implement
type TranslatorProvider interface {
	StreamCompletion(ctx context.Context, prompt string, onChunk func(string) error) error
}

// GenerativeProviderType represents the type of translation provider
type GenerativeProviderType string

const (
	ProviderOpenAI GenerativeProviderType = "openai"
	ProviderGemini GenerativeProviderType = "gemini"
)
