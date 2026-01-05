package gemini

import (
	"code-bridge/pkg/types"
	"context"
	"fmt"
	"log"

	"google.golang.org/genai"
)

type Client struct {
	client *genai.Client
}

func NewGeminiClient(geminiConfig types.GeminiConfig) *Client {
	apiKey := geminiConfig.APIKey
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create Gemini client: %v", err))
	}
	return &Client{
		client: client,
	}
}

// StreamCompletion implements streaming completion using Google Gemini API
func (c *Client) StreamCompletion(ctx context.Context, prompt string, onChunk func(string) error) error {

	stream := c.client.Models.GenerateContentStream(ctx,
		"gemini-2.5-flash",
		[]*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{
						Text: prompt,
					},
				},
			},
		},
		&genai.GenerateContentConfig{},
	)

	for chunk := range stream {
		text := chunk.Text()
		fmt.Printf("chunk: %s", text)
		err := onChunk(text)
		if err != nil {
			log.Printf("chunk failed: %v", err)
			return err
		}
	}

	fmt.Println("\n\nStream finished.")
	return nil
}
