package orient

import (
	"fmt"
	"strings"
)

// FormatComposeSummary renders concise cross-cutting patterns for orient.
func FormatComposeSummary(summary *ComposeSummary) string {
	if summary == nil {
		return ""
	}

	var b strings.Builder
	if summary.DigestPath != "" {
		b.WriteString(fmt.Sprintf("Digest available: %d clusters across %d briefs.", summary.ClustersFound, summary.BriefsComposed))
		if len(summary.Clusters) > 0 {
			top := summary.Clusters[0]
			b.WriteString(fmt.Sprintf(" Key cluster: %s (%d briefs).", top.Name, top.BriefCount))
		}
		if summary.UnprocessedBriefs > 0 {
			b.WriteString(fmt.Sprintf(" Triggered by %d unprocessed briefs.", summary.UnprocessedBriefs))
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Digest path: %s\n", summary.DigestPath))
	} else {
		b.WriteString("Cross-cutting patterns:")
		if summary.BriefsComposed > 0 {
			b.WriteString(fmt.Sprintf(" (%d briefs scanned)", summary.BriefsComposed))
		}
		if summary.UnprocessedBriefs > 0 {
			b.WriteString(fmt.Sprintf(" (%d unprocessed briefs)", summary.UnprocessedBriefs))
		}
		b.WriteString("\n")
	}

	for _, cluster := range summary.Clusters {
		b.WriteString(fmt.Sprintf("   - %s (%d briefs)\n", cluster.Name, cluster.BriefCount))
	}
	if summary.Note != "" {
		b.WriteString(fmt.Sprintf("   %s\n", summary.Note))
	}
	b.WriteString("\n")

	return b.String()
}
