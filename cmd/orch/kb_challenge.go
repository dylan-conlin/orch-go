package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/kbgate"
	"github.com/spf13/cobra"
)

var (
	kbChallengeJSON bool
)

var kbChallengeCmd = &cobra.Command{
	Use:   "challenge",
	Short: "Challenge artifact commands for adversarial gate",
}

var kbChallengeCreateCmd = &cobra.Command{
	Use:   "create <slug> <target-artifact>",
	Short: "Create a challenge artifact from template",
	Long: `Create a new challenge artifact in .kb/challenges/ with all required sections.

The challenge artifact is part of the adversarial gate pipeline. It requires:
  - Target artifact path
  - Reviewer independence metadata (3 axes)
  - Blind canonicalization findings
  - Prior-art mapping table
  - Evidence loop findings
  - Fixed severity codes (ENDOGENOUS_EVIDENCE, VOCABULARY_INFLATION,
    EXTERNAL_NOVELTY_DELTA, PUBLICATION_LANGUAGE)
  - Publication verdict (pass/fail)

Examples:
  orch kb challenge create accretion-model .kb/models/accretion/model.md
  orch kb challenge create blog-post .kb/publications/knowledge-accretion.md`,
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		targetArtifact := args[1]

		projectDir, err := os.Getwd()
		if err != nil {
			return err
		}

		path, err := kbgate.CreateChallengeArtifact(projectDir, slug, targetArtifact)
		if err != nil {
			return err
		}

		fmt.Printf("Created challenge artifact: %s\n", path)
		fmt.Printf("Target: %s\n", targetArtifact)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Fill in reviewer independence metadata")
		fmt.Println("  2. Generate blind packet: orch kb challenge packet --blind " + targetArtifact)
		fmt.Println("  3. Send to independent reviewer")
		fmt.Println("  4. Record severity codes and publication verdict")
		return nil
	},
}

var kbChallengeValidateCmd = &cobra.Command{
	Use:   "validate <challenge-path>",
	Short: "Validate a challenge artifact's structure and metadata",
	Long: `Check that a challenge artifact has valid frontmatter, reviewer independence
metadata, and severity codes.

Examples:
  orch kb challenge validate .kb/challenges/2026-03-10-accretion-model.md
  orch kb challenge validate .kb/challenges/2026-03-10-blog-post.md --json`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		challenge, err := kbgate.ParseChallengeArtifact(args[0])
		if err != nil {
			return fmt.Errorf("parse challenge: %w", err)
		}

		if err := challenge.Validate(); err != nil {
			if kbChallengeJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(map[string]any{"valid": false, "error": err.Error()})
			} else {
				fmt.Printf("✗ Challenge validation failed: %v\n", err)
			}
			return fmt.Errorf("validation failed")
		}

		if kbChallengeJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(map[string]any{"valid": true, "challenge": challenge})
		} else {
			fmt.Println("✓ Challenge artifact valid")
			fmt.Printf("  Target: %s\n", challenge.TargetArtifact)
			fmt.Printf("  Reviewer: %s (%s)\n", challenge.Reviewer.ReviewerID, challenge.Reviewer.ReviewerType)
			fmt.Printf("  Severity codes: %d\n", len(challenge.SeverityCodes))
			fmt.Printf("  Verdict: %s\n", challenge.PublicationVerdict)
		}
		return nil
	},
}

var kbChallengePacketCmd = &cobra.Command{
	Use:   "packet <artifact-path>",
	Short: "Generate a challenge packet for external review",
	Long: `Generate a blind or framed challenge packet from an artifact.

The packet contains instructions and content for an independent reviewer.
Two passes are required:

  Blind pass (--blind): Reviewer sees only canonicalized observations.
    Questions: What existing concepts? What's surprising? What's overclaim?

  Framed pass (--framed): Reviewer sees actual model/publication.
    Questions: Which are renamed concepts? Endogenous evidence? Banned terms?

Examples:
  orch kb challenge packet --blind .kb/models/accretion/model.md
  orch kb challenge packet --framed .kb/publications/knowledge-accretion.md
  orch kb challenge packet --blind .kb/models/accretion/model.md --json`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		blind, _ := cmd.Flags().GetBool("blind")
		framed, _ := cmd.Flags().GetBool("framed")

		if !blind && !framed {
			return fmt.Errorf("must specify --blind or --framed")
		}
		if blind && framed {
			return fmt.Errorf("specify only one of --blind or --framed")
		}

		var packet kbgate.ChallengePacket
		var err error

		if blind {
			packet, err = kbgate.GenerateBlindPacket(args[0])
		} else {
			packet, err = kbgate.GenerateFramedPacket(args[0])
		}
		if err != nil {
			return err
		}

		if kbChallengeJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(packet)
		}

		fmt.Printf("=== %s Challenge Packet ===\n\n", packet.PassType)
		fmt.Printf("Artifact: %s\n\n", packet.ArtifactPath)
		fmt.Printf("--- Instructions ---\n%s\n\n", packet.Instructions)
		fmt.Printf("--- Content ---\n%s\n", packet.Content)
		return nil
	},
}

func init() {
	kbChallengeValidateCmd.Flags().BoolVar(&kbChallengeJSON, "json", false, "Output as JSON")
	kbChallengePacketCmd.Flags().BoolVar(&kbChallengeJSON, "json", false, "Output as JSON")
	kbChallengePacketCmd.Flags().Bool("blind", false, "Generate blind-pass packet")
	kbChallengePacketCmd.Flags().Bool("framed", false, "Generate framed-pass packet")

	kbChallengeCmd.AddCommand(kbChallengeCreateCmd)
	kbChallengeCmd.AddCommand(kbChallengeValidateCmd)
	kbChallengeCmd.AddCommand(kbChallengePacketCmd)

	kbCmd.AddCommand(kbChallengeCmd)
}
