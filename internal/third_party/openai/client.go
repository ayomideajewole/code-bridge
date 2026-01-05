package codebridge_openai

import (
	"code-bridge/pkg/types"
	"context"
	"fmt"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
	"log"

	"github.com/openai/openai-go/v3"
)

type Client struct {
	client *openai.Client
}

func NewOpenAIClient(openAIConfig types.OpenAIConfig) *Client {
	// Create and return the client; actual SDK init may differ
	apiKey := openAIConfig.APIKey
	c := openai.NewClient(option.WithAPIKey(apiKey))
	return &Client{client: &c}
}

// StreamCompletion demonstrates a streaming call; adjust to the real SDK
func (c *Client) StreamCompletion(ctx context.Context, prompt string, onChunk func(string) error) error {
	stream := c.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Model: "gpt-5-nano",
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt)},
	})
	//stream, err := c.client.Chat.CreateStream(ctx, openai.ChatCreateParams{ /* fill */ })
	defer func(stream *ssestream.Stream[responses.ResponseStreamEventUnion]) {
		err := stream.Close()
		if err != nil {
			log.Fatalf("Failed to close stream: %v\n", err)
		}
	}(stream)

	for stream.Next() {
		currentChunk := stream.Current()
		text := currentChunk.Text
		log.Printf("chunk: %s", text)
		err := onChunk(text)
		if err != nil {
			return err
		}
	}
	// Check for any errors that occurred during streaming
	if err := stream.Err(); err != nil {
		log.Fatalf("Stream error: %v\n", err)
	}
	fmt.Println("\n\nStream finished.")
	return nil
}
