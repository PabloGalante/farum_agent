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
		Backend:  genai.BackendVertexAI,
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
	userMessage string,
	ctx domain.ConversationContext,
) (string, error) {
	// We build the prompt with your existing logic.
	prompt := BuildPrompt(userMessage, ctx)

	// System + user in Vertex format
	contents := []*genai.Content{
		genai.NewContentFromText(prompt.System, genai.RoleModel),
		genai.NewContentFromText(prompt.User, genai.RoleUser),
	}

	cfg := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr[float32](0.6),
		TopP:            genai.Ptr[float32](0.9),
		MaxOutputTokens: int32(512),
	}

	res, err := v.client.Models.GenerateContent(context.Background(), v.modelName, contents, cfg)
	if err != nil {
		return "", fmt.Errorf("vertex generate content: %w", err)
	}

	if len(res.Candidates) == 0 || res.Candidates[0].Content == nil {
		return "", fmt.Errorf("vertex returned no candidates")
	}

	var out string
	for _, part := range res.Candidates[0].Content.Parts {
		out += part.Text
	}

	return out, nil
}
