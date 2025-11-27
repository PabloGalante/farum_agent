package llm

import (
	"context"
	"fmt"

	"github.com/PabloGalante/farum-agent/internal/domain"
	"google.golang.org/genai"
)

type VertexConfig struct {
	ProjectID string
	Location  string
	ModelName string
}

type VertexClient struct {
	client    *genai.Client
	modelName string
}

// NewVertexClient creates an LLMClient based on Vertex AI (Gemini).
// Now it relies on VertexConfig instead of reading env vars directly.
func NewVertexClient(ctx context.Context, cfg VertexConfig) (*VertexClient, error) {
	if cfg.ProjectID == "" || cfg.Location == "" {
		return nil, fmt.Errorf("VertexConfig.ProjectID and Location must be set")
	}

	modelName := cfg.ModelName
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project: cfg.ProjectID,
		Location: cfg.Location,
		Backend: genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Vertex AI client: %w", err)
	}

	return &VertexClient{
		client:    client,
		modelName: modelName,
	}, nil
}

// GenerateReply implements domain.LLMClient using Vertex AI.
func (v *VertexClient) GenerateReply(
	ctx context.Context,
	userMessage string,
	convCtx domain.ConversationContext,
) (string, error) {
	// 1) System's Prompt (identity + mode)
	system := BuildSystemPrompt(convCtx.Mode)

	// 2) History (user / agent) as conversation
	var contents []*genai.Content
	for _, m := range convCtx.History {
		var role genai.Role
		switch m.Author {
		case domain.RoleUser:
			role = genai.RoleUser
		case domain.RoleAgent:
			role = genai.RoleModel
		default:
			role = genai.RoleUser
		}

		contents = append(contents, genai.NewContentFromText(m.Text, role))
	}

	// 3) Current user message
	contents = append(contents, genai.NewContentFromText(userMessage, genai.RoleUser))

	// 4) Model config
	temp := float32(0.7)
	topP := float32(0.9)
	outputTokens := int32(8192)

	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(system, genai.RoleUser),
		Temperature:       &temp,
		TopP:              &topP,
		MaxOutputTokens:   outputTokens,
	}

	// 5) Call to Vertex
	res, err := v.client.Models.GenerateContent(ctx, v.modelName, contents, cfg)
	if err != nil {
		return "", fmt.Errorf("vertex generate content: %w", err)
	}

	// 6) Extract only text
	text := res.Text()
	if text == "" {
		return "", fmt.Errorf("vertex returned empty text")
	}

	return text, nil
}
