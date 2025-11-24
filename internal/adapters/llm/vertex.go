package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/PabloGalante/farum-agent/internal/domain"
	"google.golang.org/genai"
)

type VertexClient struct {
	client    *genai.Client
	modelName string
}

// NewVertexClient creates an LLMClient based on Vertex AI (Gemini).
// Uses environment variables for project and region to simplify.
func NewVertexClient(ctx context.Context) (*VertexClient, error) {
	projectID := os.Getenv("FARUM_GCP_PROJECT")
	location := os.Getenv("FARUM_GCP_LOCATION")
	if projectID == "" || location == "" {
		return nil, fmt.Errorf("FARUM_GCP_PROJECT and FARUM_GCP_LOCATION must be set")
	}

	modelName := os.Getenv("FARUM_MODEL_NAME")
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
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
	system := BuildSystemPrompt(convCtx.Mode)

	var contents []*genai.Content

	// 1) History (user / agent)
	for _, m := range convCtx.History {
		var role genai.Role
		switch m.Author {
		case domain.RoleUser:
			role = genai.RoleUser
		case domain.RoleAgent:
			role = genai.RoleModel
		default:
			role = genai.RoleUser // Omit or handle as error
		}

		contents = append(contents, genai.NewContentFromText(m.Text, role))
	}

	// 2) Actual user message
	contents = append(contents, genai.NewContentFromText(userMessage, genai.RoleUser))
	outputTokens := int32(8192)

	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(system, genai.Role("system")),
		Temperature:       genai.Ptr[float32](0.7),
		TopP:              genai.Ptr[float32](0.9),
		MaxOutputTokens:   outputTokens,
	}

	res, err := v.client.Models.GenerateContent(ctx, v.modelName, contents, cfg)
	if err != nil {
		return "", fmt.Errorf("vertex generate content: %w", err)
	}

	if len(res.Candidates) == 0 || res.Candidates[0].Content == nil {
		return "", fmt.Errorf("vertex returned no candidates")
	}

	var out string
	for _, part := range res.Candidates[0].Content.Parts {
		out += fmt.Sprint(part)
	}

	return out, nil
}
