package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

const defaultEngine = "gpt-4o-mini"

type Client struct {
	client *openai.Client
	engine string
	debug  bool
	format string
	schema map[string]interface{}
}

func NewClient(apiKey, engine, format string, debug bool) *Client {
	if engine == "" {
		engine = defaultEngine
	}

	data, err := os.ReadFile("schema.json")
	if err != nil {
		log.Fatal(err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		log.Fatal(err)
	}

	return &Client{
		client: openai.NewClient(apiKey),
		engine: engine,
		debug:  debug,
		format: format,
		schema: schema,
	}
}

func (c *Client) Prompt(prompt, systemPrompt string, maxTokens int) (*Response, error) {
	ctx := context.Background()

	schemaJSON, err := json.Marshal(c.schema["schema"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	req := openai.ChatCompletionRequest{
		Model: c.engine,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   maxTokens,
		Temperature: 0,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "recipe_response",
				Schema: json.RawMessage(schemaJSON),
				Strict: true,
			},
		},
	}

	if c.debug {
		log.Printf("Request: %+v\n", req)
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if c.debug {
		log.Printf("Response: %+v\n", resp)
	}

	var response Response
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &response); err != nil {
		return nil, err
	}

	response.ID = resp.ID
	response.Object = resp.Object
	response.Created = resp.Created
	response.Model = resp.Model
	response.SystemFingerprint = resp.SystemFingerprint
	response.Usage = Usage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	return &response, nil
}

type Response struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	Title             string   `json:"title"`
	Date              string   `json:"date"`
	Image             string   `json:"image"`
	PrepTime          int      `json:"prepTime"`
	CookTime          int      `json:"cookTime"`
	TotalTime         int      `json:"totalTime"`
	Servings          int      `json:"servings"`
	Category          string   `json:"category"`
	Ingredients       []string `json:"ingredients"`
	Instructions      []string `json:"instructions"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	Logprobs     *string `json:"logprobs"`
	FinishReason string  `json:"finish_reason"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}