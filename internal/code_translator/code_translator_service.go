package code_translator

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"strings"
)

// ChunkType represents the type of chunk being sent
type ChunkType string

const (
	ChunkTypeExplanation ChunkType = "explanation"
	ChunkTypeNotes       ChunkType = "notes"
	ChunkTypeCode        ChunkType = "code"
	ChunkTypeError       ChunkType = "error"
	ChunkTypeRaw         ChunkType = "raw"
)

// StreamChunk represents a chunk of the translation stream
type StreamChunk struct {
	Type    ChunkType `json:"type"`
	Content string    `json:"content"`
	Delta   bool      `json:"delta,omitempty"` // true if this is a partial update
}

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

// TranslateCode sends prompt to OpenAI and streams chunks to the callback
func (s *CodeTranslatorService) TranslateCode(ctx context.Context, code, sourceLang, targetLang string, onChunk func(string) error) error {
	prompt := buildPrompt(code, sourceLang, targetLang)

	s.logger.Info("translating code",
		zap.String("source_language", sourceLang),
		zap.String("target_language", targetLang),
	)

	// Stream handler that processes chunks in real-time
	var fullResponse strings.Builder
	currentSection := ""
	sectionBuffer := strings.Builder{}

	err := s.provider.StreamCompletion(ctx, prompt, func(chunk string) error {
		fullResponse.WriteString(chunk)
		text := fullResponse.String()

		// Detect section changes
		newSection := detectCurrentSection(text)

		// If section changed, send the complete previous section
		if newSection != currentSection && currentSection != "" {
			content := extractSectionContent(fullResponse.String(), currentSection)
			if content != "" {
				streamChunk := StreamChunk{
					Type:    ChunkType(currentSection),
					Content: content,
					Delta:   false,
				}
				jsonData, _ := json.Marshal(streamChunk)
				if err := onChunk(string(jsonData)); err != nil {
					return err
				}
			}
			sectionBuffer.Reset()
		}

		currentSection = newSection

		// Send delta updates for current section
		if currentSection != "" {
			content := extractSectionContent(text, currentSection)
			if content != "" && content != sectionBuffer.String() {
				streamChunk := StreamChunk{
					Type:    ChunkType(currentSection),
					Content: content,
					Delta:   true,
				}
				jsonData, _ := json.Marshal(streamChunk)
				if err := onChunk(string(jsonData)); err != nil {
					return err
				}
				sectionBuffer.WriteString(content)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Send final complete sections
	return s.sendFinalSections(fullResponse.String(), onChunk)
}

func detectCurrentSection(text string) string {
	// Check which section we're currently in based on the last header seen
	lastExplanation := strings.LastIndex(strings.ToLower(text), "=== explanation ===")
	lastNotes := strings.LastIndex(strings.ToLower(text), "=== translation notes ===")
	lastCode := strings.LastIndex(strings.ToLower(text), "=== translated code ===")

	// Find the most recent section header
	if lastCode > lastNotes && lastCode > lastExplanation {
		return "code"
	} else if lastNotes > lastExplanation && lastNotes > lastCode {
		return "notes"
	} else if lastExplanation >= 0 {
		return "explanation"
	}

	return ""
}

func extractSectionContent(text, section string) string {
	lowerText := strings.ToLower(text)

	switch section {
	case "explanation":
		start := strings.Index(lowerText, "=== explanation ===")
		if start == -1 {
			return ""
		}
		start += len("=== explanation ===")

		// Find end (next section or end of text)
		end := strings.Index(lowerText[start:], "=== translation notes ===")
		if end == -1 {
			end = len(text) - start
		}

		return strings.TrimSpace(text[start : start+end])

	case "notes":
		start := strings.Index(lowerText, "=== translation notes ===")
		if start == -1 {
			return ""
		}
		start += len("=== translation notes ===")

		end := strings.Index(lowerText[start:], "=== translated code ===")
		if end == -1 {
			end = len(text) - start
		}

		return strings.TrimSpace(text[start : start+end])

	case "code":
		start := strings.Index(lowerText, "=== translated code ===")
		if start == -1 {
			return ""
		}
		start += len("=== translated code ===")

		content := strings.TrimSpace(text[start:])
		// Remove markdown code fences
		content = strings.TrimPrefix(content, "```javascript")
		content = strings.TrimPrefix(content, "```typescript")
		content = strings.TrimPrefix(content, "```python")
		content = strings.TrimPrefix(content, "```go")
		content = strings.TrimPrefix(content, "```rust")
		content = strings.TrimPrefix(content, "```java")
		content = strings.TrimPrefix(content, "```csharp")
		content = strings.TrimPrefix(content, "```cpp")
		content = strings.TrimPrefix(content, "```php")
		content = strings.TrimPrefix(content, "```ruby")
		content = strings.TrimPrefix(content, "```swift")
		content = strings.TrimPrefix(content, "```kotlin")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")

		return strings.TrimSpace(content)
	}

	return ""
}

func (s *CodeTranslatorService) sendFinalSections(text string, onChunk func(string) error) error {
	// Send final complete versions of all sections
	sections := []string{"explanation", "notes", "code"}

	for _, section := range sections {
		content := extractSectionContent(text, section)
		if content != "" {
			chunk := StreamChunk{
				Type:    ChunkType(section),
				Content: content,
				Delta:   false,
			}
			jsonData, _ := json.Marshal(chunk)
			if err := onChunk(string(jsonData)); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildPrompt(code, source, target string) string {
	b := strings.Builder{}
	b.WriteString("You are a code translator. You MUST respond in the EXACT format shown below.\n\n")
	b.WriteString("CRITICAL: You must include ALL THREE sections in your response:\n")
	b.WriteString("1. === EXPLANATION ===\n")
	b.WriteString("2. === TRANSLATION NOTES ===\n")
	b.WriteString("3. === TRANSLATED CODE ===\n\n")

	if source != "" {
		b.WriteString(fmt.Sprintf("Translate this %s code to %s.\n\n", source, target))
	} else {
		b.WriteString(fmt.Sprintf("Translate this code to %s.\n\n", target))
	}

	b.WriteString("Your response MUST follow this EXACT structure:\n\n")
	b.WriteString("=== EXPLANATION ===\n")
	b.WriteString("[Write 2-3 sentences explaining what the original code does]\n\n")
	b.WriteString("=== TRANSLATION NOTES ===\n")
	b.WriteString("- [Key difference 1 between source and target language]\n")
	b.WriteString("- [Key difference 2 between source and target language]\n")
	b.WriteString("- [Key difference 3 between source and target language]\n\n")
	b.WriteString("=== TRANSLATED CODE ===\n")
	b.WriteString("```" + target + "\n")
	b.WriteString("[The complete translated code goes here]\n")
	b.WriteString("```\n\n")
	b.WriteString("SOURCE CODE TO TRANSLATE:\n")
	b.WriteString("```" + source + "\n")
	b.WriteString(code)
	b.WriteString("\n```\n\n")
	b.WriteString("IMPORTANT: You MUST include all three sections (EXPLANATION, TRANSLATION NOTES, and TRANSLATED CODE) in your response. Do not skip any section.")

	return b.String()
}
