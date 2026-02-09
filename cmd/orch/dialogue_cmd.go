package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/dialogue"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/spf13/cobra"
)

var (
	dialogueQuestionerModel string
	dialogueExpertModel     string
	dialogueExploreTurns    int
	dialogueConvergeTurns   int
	dialogueMaxTurns        int
	dialogueMaxTokens       int
	dialoguePollInterval    time.Duration
	dialogueResponseTimeout time.Duration
	dialogueVerbose         bool
)

var dialogueCmd = &cobra.Command{
	Use:   "dialogue [identifier] [topic]",
	Short: "Run Ghost Partner dialogue against a running agent",
	Long: `Run a turn-based Ghost Partner dialogue with a running OpenCode agent.

The Ghost Partner is a tool-free Anthropic Messages API client that asks probing
questions and relays them to a running agent session. The relay loop follows a
3-phase director flow:
  - Explore: probing questions only
  - Converge: synthesis and decision pressure
  - Terminate: verdict-driven finish with [VERDICT: ...] token

identifier accepts the same values as orch send:
  - session ID (ses_xxx)
  - beads ID (project-xxxx)
  - workspace name`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		topic := strings.TrimSpace(strings.Join(args[1:], " "))
		return runDialogue(opencode.NewClient(serverURL), identifier, topic, os.Stdout)
	},
}

func init() {
	dialogueCmd.Flags().StringVar(&dialogueQuestionerModel, "questioner-model", "sonnet", "Ghost Partner model (Anthropic only)")
	dialogueCmd.Flags().StringVar(&dialogueExpertModel, "expert-model", "", "Override running agent model for relayed prompts")
	dialogueCmd.Flags().IntVar(&dialogueExploreTurns, "explore-turns", 6, "Turns in Explore phase")
	dialogueCmd.Flags().IntVar(&dialogueConvergeTurns, "converge-turns", 12, "Turns in Converge phase")
	dialogueCmd.Flags().IntVar(&dialogueMaxTurns, "max-turns", 15, "Hard turn cap")
	dialogueCmd.Flags().IntVar(&dialogueMaxTokens, "max-tokens", dialogue.DefaultMaxTokens, "Max output tokens per Ghost Partner turn")
	dialogueCmd.Flags().DurationVar(&dialoguePollInterval, "poll-interval", 2*time.Second, "Expert response poll interval")
	dialogueCmd.Flags().DurationVar(&dialogueResponseTimeout, "response-timeout", 2*time.Minute, "Expert response timeout per turn")
	dialogueCmd.Flags().BoolVar(&dialogueVerbose, "verbose", true, "Print turn-by-turn relay output")
}

func runDialogue(client opencode.ClientInterface, identifier, topic string, out io.Writer) error {
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("topic is required")
	}

	sessionID, err := resolveSessionIDWithClient(client, identifier)
	if err != nil {
		return fmt.Errorf("resolve session: %w", err)
	}

	questionerModelID, err := resolveDialogueQuestionerModel(dialogueQuestionerModel)
	if err != nil {
		return err
	}

	expertModel := ""
	if strings.TrimSpace(dialogueExpertModel) != "" {
		expertModel = model.Resolve(dialogueExpertModel).Format()
	}

	questioner, err := dialogue.NewClient(dialogue.Config{Model: questionerModelID})
	if err != nil {
		return fmt.Errorf("create Ghost Partner client: %w", err)
	}

	workspacePath := resolveDialogueWorkspacePath(client, sessionID)

	logger := events.NewLogger(events.DefaultLogPath())
	_ = logger.Log(events.Event{
		Type:      "dialogue.started",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"identifier":       identifier,
			"topic":            topic,
			"questioner_model": questionerModelID,
			"expert_model":     expertModel,
		},
	})

	if dialogueVerbose {
		fmt.Fprintf(out, "Starting Ghost Partner dialogue with %s\n", sessionID)
		fmt.Fprintf(out, "Topic: %s\n", topic)
		fmt.Fprintf(out, "Phases: explore=%d converge=%d max=%d\n", dialogueExploreTurns, dialogueConvergeTurns, dialogueMaxTurns)
	}

	observer := dialogue.Observer(dialogue.NoopObserver{})
	if dialogueVerbose {
		observer = &relayWriterObserver{out: out}
	}

	relayCfg := dialogue.RelayConfig{
		Topic:           topic,
		QuestionerModel: questionerModelID,
		ExpertModel:     expertModel,
		ExploreTurns:    dialogueExploreTurns,
		ConvergeTurns:   dialogueConvergeTurns,
		MaxTurns:        dialogueMaxTurns,
		MaxTokens:       dialogueMaxTokens,
		PollInterval:    dialoguePollInterval,
		ResponseTimeout: dialogueResponseTimeout,
	}

	startedAt := time.Now().UTC()
	result, err := dialogue.RunRelay(context.Background(), questioner, client, sessionID, relayCfg, observer)
	completedAt := time.Now().UTC()
	if err != nil {
		_ = logger.Log(events.Event{
			Type:      "dialogue.error",
			SessionID: sessionID,
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return err
	}

	if strings.TrimSpace(workspacePath) != "" {
		artifacts, artifactErr := dialogue.WriteArtifacts(workspacePath, dialogue.TranscriptMetadata{
			SessionID:       sessionID,
			Topic:           topic,
			QuestionerModel: questionerModelID,
			ExpertModel:     expertModel,
			StartedAt:       startedAt,
			CompletedAt:     completedAt,
		}, relayCfg, result)
		if artifactErr != nil {
			_ = logger.Log(events.Event{
				Type:      "dialogue.artifacts.error",
				SessionID: sessionID,
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"error":          artifactErr.Error(),
					"workspace_path": workspacePath,
				},
			})
			fmt.Fprintf(os.Stderr, "Warning: failed to export dialogue artifacts: %v\n", artifactErr)
		} else {
			_ = logger.Log(events.Event{
				Type:      "dialogue.artifacts.generated",
				SessionID: sessionID,
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"workspace_path":    workspacePath,
					"transcript":        artifacts.TranscriptPath,
					"artifact_markdown": artifacts.ArtifactMDPath,
					"artifact_json":     artifacts.ArtifactJSONPath,
					"decision_count":    artifacts.DecisionCount,
					"follow_up_count":   artifacts.FollowUpCount,
				},
			})

			fmt.Fprintf(out, "\nSaved dialogue transcript: %s\n", artifacts.TranscriptPath)
			fmt.Fprintf(out, "Saved derived artifacts: %s\n", artifacts.ArtifactMDPath)
			fmt.Fprintf(out, "Saved structured artifacts: %s\n", artifacts.ArtifactJSONPath)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Warning: session workspace not found; skipping dialogue artifact export")
	}

	_ = logger.Log(events.Event{
		Type:      "dialogue.completed",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"turns":      len(result.Turns),
			"approved":   result.Approved,
			"verdict":    result.Verdict,
			"end_reason": result.EndReason,
		},
	})

	if result.Approved {
		fmt.Fprintf(out, "\nDialogue complete: approved (%s) after %d turns\n", result.Verdict, len(result.Turns))
		return nil
	}

	fmt.Fprintf(out, "\nDialogue ended without approval (%s) after %d turns\n", result.EndReason, len(result.Turns))
	return nil
}

func resolveDialogueQuestionerModel(spec string) (string, error) {
	resolved := model.Resolve(strings.TrimSpace(spec))
	if resolved.Provider != "anthropic" {
		return "", fmt.Errorf("questioner model must be Anthropic for Messages API; got %s", resolved.Format())
	}
	return resolved.ModelID, nil
}

type relayWriterObserver struct {
	out io.Writer
}

func (o *relayWriterObserver) OnGhostTurn(turn int, phase dialogue.Phase, text string) {
	fmt.Fprintf(o.out, "\n[Turn %d][%s] Ghost Partner\n%s\n", turn, phase, text)
}

func (o *relayWriterObserver) OnExpertTurn(turn int, text string) {
	fmt.Fprintf(o.out, "[Turn %d] Agent Response\n%s\n", turn, text)
}

func resolveDialogueWorkspacePath(client opencode.ClientInterface, sessionID string) string {
	if session, err := client.GetSession(sessionID); err == nil && session != nil {
		if directory := strings.TrimSpace(session.Directory); directory != "" {
			return directory
		}
	}

	projectDir, err := currentProjectDir()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(findWorkspaceBySessionID(projectDir, sessionID))
}
