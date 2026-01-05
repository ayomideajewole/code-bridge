package code_translator

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"strings"
)

// TranslatorProviderInterface defines the methods required for translation providers
type TranslatorProviderInterface interface {
	StreamCompletion(ctx context.Context, prompt string, onChunk func(string) error) error
}

// CodeTranslatorService provides code translation functionalities
type CodeTranslatorService struct {
	logger   *zap.Logger
	provider TranslatorProviderInterface
}

// NewCodeTranslatorService creates a new instance of CodeTranslatorService
func NewCodeTranslatorService(logger *zap.Logger, provider TranslatorProviderInterface) *CodeTranslatorService {
	return &CodeTranslatorService{
		logger:   logger,
		provider: provider,
	}
}

// TranslateCode translates code from one programming language to another
//func (s *CodeTranslatorService) TranslateCode(code, sourceLang, targetLang string) (string, error) {
//	s.logger.Info("translating code",
//		zap.String("source_language", sourceLang),
//		zap.String("target_language", targetLang),
//	)
//
//	// Placeholder for actual translation logic
//	translatedCode := "// Translated code\n" + code
//
//	s.logger.Info("code translation completed")
//	return translatedCode, nil
//}

// TranslateCode sends prompt to OpenAI and streams chunks to the callback
func (s *CodeTranslatorService) TranslateCode(ctx context.Context, code, sourceLang, targetLang string, onChunk func(string) error) error {
	prompt := buildPrompt(code, sourceLang, targetLang)

	s.logger.Info("translating code",
		zap.String("source_language", sourceLang),
		zap.String("target_language", targetLang),
	)

	// use SDK streaming helper
	err := s.provider.StreamCompletion(ctx, prompt, func(chunk string) error {
		// Here we can try to accumulate JSON or pass raw text depending on model output
		// For demo, forward raw chunk
		return onChunk(chunk)
	})
	if err != nil {
		return err
	}
	return nil
}

func buildPrompt(code, source, target string) string {
	b := strings.Builder{}
	b.WriteString("You are a helpful code translator.\n")
	b.WriteString(fmt.Sprintf("Source language: %s\n", source))
	b.WriteString(fmt.Sprintf("Target language: %s\n", target))
	b.WriteString("Given the following SOURCE_CODE, first explain what it does in plain English, then provide the TRANSLATED_CODE. Respond with a JSON object with keys 'explanation' and 'code'.\n")
	b.WriteString("SOURCE_CODE:\n```")
	b.WriteString(code)
	b.WriteString("\n```")
	return b.String()
}
