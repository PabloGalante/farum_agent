package llm

import (
	"context"
	"fmt"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

type MockLLM struct{}

func NewMockLLM() *MockLLM {
	return &MockLLM{}
}

func (m *MockLLM) GenerateReply(ctx context.Context, prompt string, convCtx domain.ConversationContext) (string, error) {
	// Here we could use minimun rules to give Farum some personality
	return fmt.Sprintf("Te escucho. Dijiste %q. Contame un poco mas sobre ocmo te hace sentir eso", prompt), nil
}
