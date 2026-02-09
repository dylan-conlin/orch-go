package episodic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

type Context struct {
	Boundary        string
	Project         string
	Workspace       string
	SessionID       string
	BeadsID         string
	EvidencePointer string
	Now             time.Time
}

func ExtractEvent(event events.Event, ctx Context) (*ActionMemory, error) {
	if strings.TrimSpace(event.Type) == "" {
		return nil, nil
	}

	now := ctx.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	action, outcome, confidence, ok := mapEvent(event)
	if !ok {
		return nil, nil
	}

	boundary := strings.TrimSpace(ctx.Boundary)
	if boundary == "" {
		boundary = boundaryForEvent(event.Type)
	}

	evidencePointer := strings.TrimSpace(ctx.EvidencePointer)
	if evidencePointer == "" {
		evidencePointer = fmt.Sprintf("~/.orch/events.jsonl#type=%s#timestamp=%d", event.Type, event.Timestamp)
	}

	timestamp := event.Timestamp
	if timestamp == 0 {
		timestamp = now.Unix()
	}

	memory := ActionMemory{
		Boundary:  boundary,
		Project:   ctx.Project,
		Workspace: ctx.Workspace,
		SessionID: firstNonEmpty(ctx.SessionID, event.SessionID),
		BeadsID:   firstNonEmpty(ctx.BeadsID, stringField(event.Data, "beads_id")),
		Action:    action,
		Outcome:   outcome,
		Evidence: Evidence{
			Kind:      EvidenceKindEventsJSONL,
			Pointer:   evidencePointer,
			Timestamp: timestamp,
			Hash:      hashValue(event),
		},
		Confidence: confidence,
		CreatedAt:  now,
	}
	memory.SetDefaults(now)
	if err := memory.Validate(); err != nil {
		return nil, err
	}

	return &memory, nil
}

func ExtractActivityParts(parts []activity.MessagePartResponse, ctx Context) ([]ActionMemory, error) {
	now := ctx.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var result []ActionMemory
	for _, item := range parts {
		part := item.Properties.Part
		if part.Type != "tool" {
			continue
		}

		status := OutcomeObserved
		summary := summarizeToolPart(part)
		confidence := 0.75
		input := "-"

		if part.State != nil {
			status = firstNonEmpty(strings.TrimSpace(part.State.Status), OutcomeObserved)
			if part.State.Input != nil {
				input = toInput(part.State.Input)
			}
			if strings.EqualFold(status, "completed") || strings.EqualFold(status, "success") {
				status = OutcomeSuccess
				confidence = 0.85
			}
			if strings.EqualFold(status, "failed") || strings.EqualFold(status, "error") {
				status = OutcomeFailed
				confidence = 0.9
			}
		}

		ts := item.Timestamp
		if ts == 0 {
			ts = now.Unix()
		}

		pointer := strings.TrimSpace(ctx.EvidencePointer)
		if pointer == "" {
			workspace := strings.TrimSpace(ctx.Workspace)
			if workspace == "" {
				workspace = "unknown"
			}
			pointer = fmt.Sprintf("~/.orch/workspace/%s/ACTIVITY.json", workspace)
		}
		pointer = fmt.Sprintf("%s#message=%s&part=%s", pointer, item.Properties.MessageID, part.ID)

		memory := ActionMemory{
			Boundary:  firstNonEmpty(ctx.Boundary, BoundaryCompletion),
			Project:   ctx.Project,
			Workspace: ctx.Workspace,
			SessionID: firstNonEmpty(ctx.SessionID, part.SessionID),
			BeadsID:   ctx.BeadsID,
			Action: Action{
				Type:  "tool",
				Name:  firstNonEmpty(part.Tool, "tool"),
				Input: input,
			},
			Outcome: Outcome{
				Status:  status,
				Summary: summary,
			},
			Evidence: Evidence{
				Kind:      EvidenceKindActivityJSON,
				Pointer:   pointer,
				Timestamp: ts,
				Hash:      hashValue(item),
			},
			Confidence: confidence,
			CreatedAt:  now,
		}
		memory.SetDefaults(now)
		if err := memory.Validate(); err != nil {
			return nil, err
		}
		result = append(result, memory)
	}

	return result, nil
}

func mapEvent(event events.Event) (Action, Outcome, float64, bool) {
	switch event.Type {
	case events.EventTypeSessionSpawned:
		return Action{Type: "lifecycle", Name: event.Type, Input: toInput(event.Data)}, Outcome{Status: OutcomeSuccess, Summary: firstNonEmpty(fmt.Sprintf("Session spawned (%s)", stringField(event.Data, "spawn_mode")), "Session spawned")}, 0.95, true
	case "session.send":
		method := stringField(event.Data, "method")
		summary := "Command sent"
		if strings.TrimSpace(method) != "" {
			summary = fmt.Sprintf("Command sent via %s", method)
		}
		return Action{Type: "command", Name: event.Type, Input: firstNonEmpty(stringField(event.Data, "message"), toInput(event.Data))}, Outcome{Status: OutcomeSuccess, Summary: summary}, 0.85, true
	case "session.phase":
		phase := firstNonEmpty(stringField(event.Data, "phase"), "unknown")
		summary := firstNonEmpty(stringField(event.Data, "summary"), fmt.Sprintf("Phase updated to %s", phase))
		return Action{Type: "phase", Name: phase, Input: firstNonEmpty(stringField(event.Data, "summary"), "-")}, Outcome{Status: OutcomeReported, Summary: summary}, 0.9, true
	case events.EventTypeVerificationFailed:
		gates := toInput(event.Data["gates_failed"])
		summary := firstNonEmpty(firstFromSlice(event.Data, "errors"), "Verification failed")
		return Action{Type: "verification", Name: "gates_failed", Input: gates}, Outcome{Status: OutcomeFailed, Summary: summary}, 0.95, true
	case events.EventTypeVerificationBypassed:
		gate := firstNonEmpty(stringField(event.Data, "gate"), "unknown")
		reason := firstNonEmpty(stringField(event.Data, "reason"), "No reason provided")
		return Action{Type: "verification", Name: fmt.Sprintf("bypass.%s", gate), Input: reason}, Outcome{Status: OutcomeBypassed, Summary: fmt.Sprintf("Bypassed %s", gate)}, 0.9, true
	case "verification.outcome":
		passed := boolField(event.Data, "passed")
		if passed {
			return Action{Type: "verification", Name: "gates", Input: "all"}, Outcome{Status: OutcomeSuccess, Summary: "Verification passed"}, 0.95, true
		}
		gates := toInput(event.Data["gates_failed"])
		summary := firstNonEmpty(firstFromSlice(event.Data, "errors"), "Verification failed")
		return Action{Type: "verification", Name: "gates", Input: gates}, Outcome{Status: OutcomeFailed, Summary: summary}, 0.95, true
	case "completion.activity_export":
		status := firstNonEmpty(stringField(event.Data, "status"), OutcomeSuccess)
		summary := firstNonEmpty(stringField(event.Data, "summary"), "Activity export completed")
		return Action{Type: "completion", Name: "activity_export", Input: firstNonEmpty(stringField(event.Data, "path"), "-")}, Outcome{Status: status, Summary: summary}, 0.9, true
	default:
		return Action{}, Outcome{}, 0, false
	}
}

func boundaryForEvent(kind string) string {
	switch kind {
	case events.EventTypeSessionSpawned:
		return BoundarySpawn
	case "session.send", "session.phase":
		return BoundaryCommand
	case events.EventTypeVerificationFailed, events.EventTypeVerificationBypassed, "verification.outcome":
		return BoundaryVerification
	case "completion.activity_export", events.EventTypeAgentCompleted:
		return BoundaryCompletion
	default:
		return BoundaryCommand
	}
}

func summarizeToolPart(part activity.PartDetails) string {
	if part.State != nil {
		if strings.TrimSpace(part.State.Title) != "" {
			return part.State.Title
		}
		if strings.TrimSpace(part.State.Output) != "" {
			return firstLine(part.State.Output)
		}
	}
	if strings.TrimSpace(part.Text) != "" {
		return firstLine(part.Text)
	}
	if strings.TrimSpace(part.Tool) != "" {
		return fmt.Sprintf("Tool invocation: %s", part.Tool)
	}
	return "Tool invocation observed"
}

func firstLine(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	idx := strings.IndexByte(v, '\n')
	if idx == -1 {
		return v
	}
	if idx == 0 {
		return "-"
	}
	return strings.TrimSpace(v[:idx])
}

func hashValue(v interface{}) string {
	raw, err := json.Marshal(v)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", v))
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func toInput(v interface{}) string {
	if v == nil {
		return "-"
	}
	switch value := v.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return "-"
		}
		return value
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		if len(raw) == 0 {
			return "-"
		}
		return string(raw)
	}
}

func stringField(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func boolField(data map[string]interface{}, key string) bool {
	if data == nil {
		return false
	}
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)
	if ok {
		return b
	}
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(s), "true")
}

func firstFromSlice(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) == 0 {
		return ""
	}
	s, ok := arr[0].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
