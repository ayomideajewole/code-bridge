package translator_provider

import (
	"code-bridge/internal/third_party/gemini"
	codebridge_openai "code-bridge/internal/third_party/openai"
	"code-bridge/pkg/types"
	"fmt"
)

// Factory creates translator providers based on the specified type
type Factory struct {
	config *types.Config
}

// NewFactory creates a new provider factory
func NewFactory(config *types.Config) *Factory {
	return &Factory{
		config: config,
	}
}

// CreateProvider creates a translator provider based on the specified type
func (f *Factory) CreateProvider(providerType GenerativeProviderType) (TranslatorProvider, error) {
	switch providerType {
	case ProviderOpenAI:
		return codebridge_openai.NewOpenAIClient(f.config.OpenAI), nil
	case ProviderGemini:
		return gemini.NewGeminiClient(f.config.Gemini), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
