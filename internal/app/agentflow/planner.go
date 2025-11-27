package agentflow

import (
	"context"
	"fmt"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

// PlannerAgent: transforms the clarified problem into a concrete action plan.
type PlannerAgent struct {
	llm domain.LLMClient
}

func NewPlannerAgent(llm domain.LLMClient) *PlannerAgent {
	return &PlannerAgent{llm: llm}
}

func (a *PlannerAgent) Name() string {
	return "planner"
}

func (a *PlannerAgent) Run(ctx context.Context, in AgentInput) (AgentOutput, error) {
	prompt := fmt.Sprintf(
		"You are Farum's Planner agent. The Listener agent has clarified the user's concern.\n"+
			"Now your job is to create a short, concrete action plan with 2-4 steps that the user can follow.\n"+
			"Be realistic, kind and practical.\n\nPrevious agent output:\n%s",
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
