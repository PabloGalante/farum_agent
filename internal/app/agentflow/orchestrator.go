package agentflow

import (
	"context"
	"fmt"
	"time"

	"github.com/PabloGalante/farum-agent/internal/app/tools"
	"github.com/PabloGalante/farum-agent/internal/domain"
	"github.com/PabloGalante/farum-agent/internal/observability"
)

// Orchestrator is responsible for running multiple agents in sequence.
type Orchestrator struct {
	llm         domain.LLMClient
	journalTool tools.Tool
	agents      []Agent
}

// NewDefaultOrchestrator constructs a flow with Listener -> Planner -> Reflector.
func NewDefaultOrchestrator(llm domain.LLMClient, journalTool tools.Tool) *Orchestrator {
	return &Orchestrator{
		llm:         llm,
		journalTool: journalTool,
		agents: []Agent{
			NewListenerAgent(llm),
			NewPlannerAgent(llm),
			NewReflectorAgent(llm, journalTool),
		},
	}
}

// Run executes the chain of agents sequentially.
func (o *Orchestrator) Run(
	ctx context.Context,
	userMessage string,
	convCtx domain.ConversationContext,
) (string, error) {
	if len(o.agents) == 0 {
		return "", fmt.Errorf("no agents configured in orchestrator")
	}

	log := observability.LoggerFromContext(ctx).With(
		"session_id", convCtx.SessionID,
		"user_id", convCtx.UserID,
	)
	log.Info("Orchestrator started", "agents_count", len(o.agents))

	in := AgentInput{
		UserMessage: userMessage,
		ConvCtx:     convCtx,
	}

	var (
		out AgentOutput
		err error
	)

	for _, ag := range o.agents {
		start := time.Now()
		log.Info("agent run start", "agent", ag.Name())

		out, err = ag.Run(ctx, in)
		if err != nil {
			log.Error("agent failed",
				"agent", ag.Name(),
				"error", err)
			return "", fmt.Errorf("agent %s failed: %w", ag.Name(), err)
		}

		elapsed := time.Since(start)
		log.Info("agent rund end", "agent", ag.Name(), "elapsed_ms", elapsed.Milliseconds())

		// The output of an agent is the input for the next agent
		in.UserMessage = out.Reply
		in.ConvCtx = out.UpdatedContext
	}

	// Return the last generated response
	log.Info("orchestrator end")
	return out.Reply, nil
}
