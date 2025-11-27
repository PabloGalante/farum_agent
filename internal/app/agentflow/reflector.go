package agentflow

import (
	"context"
	"fmt"

	"github.com/PabloGalante/farum-agent/internal/app/tools"
	"github.com/PabloGalante/farum-agent/internal/domain"
	"github.com/PabloGalante/farum-agent/internal/observability"
)

// ReflectorAgent: helps close the interaction with a brief reflection.
type ReflectorAgent struct {
	llm         domain.LLMClient
	journalTool tools.Tool
}

func NewReflectorAgent(llm domain.LLMClient, journalTool tools.Tool) *ReflectorAgent {
	return &ReflectorAgent{
		llm:         llm,
		journalTool: journalTool,
	}
}

func (a *ReflectorAgent) Name() string {
	return "reflector"
}

func (a *ReflectorAgent) Run(ctx context.Context, in AgentInput) (AgentOutput, error) {
	log := observability.LoggerFromContext(ctx).With("agent", a.Name())
	log.Info("reflector agent running")

	prompt := fmt.Sprintf(
		"You are Farum's Reflector agent. The Planner agent proposed an action plan.\n"+
			"Your job is to close the conversation with a short reflective message that helps the user\n"+
			"connect emotionally with the plan, and maybe ask 1 gentle question for journaling.\n\n"+
			"Previous agent output:\n%s",
		in.UserMessage,
	)

	reply, err := a.llm.GenerateReply(ctx, prompt, in.ConvCtx)
	if err != nil {
		log.Error("reflector agent error", "error", err)
		return AgentOutput{}, err
	}

	updatedCtx := in.ConvCtx
	if a.journalTool != nil {
		tctx := tools.ToolContext{
			UserID:    string(in.ConvCtx.UserID),
			SessionID: string(in.ConvCtx.SessionID),
			RequestID: "",
		}

		// MVP
		input := map[string]any{
			"problem_summary": "",
			"reflection":      reply,
			"mood_before":     "",
			"mood_after":      "",
			"actions":         []any{},
		}

		_, _ = a.journalTool.Call(ctx, tctx, input)
	}

	log.Info("reflector agent success")
	return AgentOutput{
		Reply:          reply,
		UpdatedContext: updatedCtx,
	}, nil
}
