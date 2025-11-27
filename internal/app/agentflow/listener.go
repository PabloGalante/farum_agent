package agentflow

import (
	"context"
	"fmt"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

// ListenerAgent: focuses on listening and clarifying the user's problem.
type ListenerAgent struct {
	llm domain.LLMClient
}

func NewListenerAgent(llm domain.LLMClient) *ListenerAgent {
	return &ListenerAgent{llm: llm}
}

func (a *ListenerAgent) Name() string {
	return "listener"
}

func (a *ListenerAgent) Run(ctx context.Context, in AgentInput) (AgentOutput, error) {
	prompt := fmt.Sprintf(
		"You are Farum's Listener agent. Your job is to carefully listen, clarify the user's concern, and restate it in a clear, empathetic way.\n\nUser: %s",
		in.UserMessage,
	)

	reply, err := a.llm.GenerateReply(ctx, prompt, in.ConvCtx)
	if err != nil {
		return AgentOutput{}, err
	}

	return AgentOutput{
		Reply:          reply,
		UpdatedContext: in.ConvCtx,
	}, nil
}
