package dialogue

import (
	"fmt"
	"strings"
)

// Phase defines the dialogue director phase.
type Phase string

const (
	PhaseExplore   Phase = "explore"
	PhaseConverge  Phase = "converge"
	PhaseTerminate Phase = "terminate"
)

func phaseForTurn(turn, exploreTurns, convergeTurns int) Phase {
	if turn <= exploreTurns {
		return PhaseExplore
	}
	if turn <= convergeTurns {
		return PhaseConverge
	}
	return PhaseTerminate
}

func buildGhostSystemPrompt(topic string, phase Phase, turn, maxTurns int) string {
	var phaseRules string
	switch phase {
	case PhaseExplore:
		phaseRules = strings.TrimSpace(`
Phase 1 (Explore):
- Ask exactly one focused probing question.
- Do NOT propose solutions.
- Do NOT include a verdict token.
- Goal: reduce ambiguity and surface hidden assumptions.`)
	case PhaseConverge:
		phaseRules = strings.TrimSpace(`
Phase 2 (Converge):
- Synthesize what you've learned and force concrete decisions.
- Ask one decision-driving question OR provide a concise recommendation.
- If you judge the design direction is ready, include a verdict token:
  [VERDICT: APPROVED]
- If approved, include:
  ## Decision
  <one concise decision statement>
  ## Follow-Up Issues
  - <issue draft or action item>
- Otherwise include:
  [VERDICT: CONTINUE]`)
	default:
		phaseRules = strings.TrimSpace(`
Phase 3 (Terminate):
- Produce a final judgment.
- Include exactly one verdict token:
  [VERDICT: APPROVED] or [VERDICT: CONTINUE]
- If approved, include:
  ## Decision
  <one concise decision statement>
  ## Follow-Up Issues
  - <issue draft or action item>
- Keep your message concise and explicit.`)
	}

	return fmt.Sprintf(strings.TrimSpace(`
You are the Ghost Partner in a design dialogue.

Topic: %s
Turn: %d/%d

You have no tool access and no direct codebase access.
Your only input is the expert's replies.

Rules:
- Be direct and specific.
- Keep each turn to one message.
- Prefer short, high-leverage questions.

%s
`), topic, turn, maxTurns, phaseRules)
}

func initialGhostUserMessage(topic string) Message {
	return Message{
		Role: "user",
		Content: fmt.Sprintf(
			"Start the Ghost Partner dialogue for topic: %s\nAsk your first question.",
			topic,
		),
	}
}
