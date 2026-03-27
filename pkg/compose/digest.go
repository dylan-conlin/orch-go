package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Digest represents a composed digest of briefs.
type Digest struct {
	Date            time.Time
	BriefsComposed  int
	ClustersFound   int
	Clusters        []*DigestCluster
	Unclustered     []*Brief
	TensionOrphans  []TensionEntry
	EpistemicStatus string
}

// DigestCluster combines a Cluster with its thread matches.
type DigestCluster struct {
	*Cluster
	ThreadMatches []ThreadMatch
}

// TensionEntry pairs a brief ID with its tension text.
type TensionEntry struct {
	BriefID string
	Text    string
}

// DefaultEpistemicStatus is the standard label for unverified clustering.
const DefaultEpistemicStatus = `This digest clusters briefs by observed content similarity.
It has NOT been verified by a human. Clusters may be:
- Coincidental (briefs happen to use similar words)
- Incomplete (the real pattern includes briefs not in this cluster)
- Wrong (the briefs are related, but not for the reason shown)`

// MaxDocumentFrequency is the fraction of briefs a keyword can appear in before
// it's considered too common to be discriminative. Words in >20% of briefs are filtered.
const MaxDocumentFrequency = 0.20

// Compose runs the full composition pipeline: load briefs, cluster, match threads, build digest.
func Compose(briefsDir, threadsDir string) (*Digest, error) {
	briefs, err := LoadBriefs(briefsDir)
	if err != nil {
		return nil, fmt.Errorf("loading briefs: %w", err)
	}

	if len(briefs) == 0 {
		return nil, fmt.Errorf("no briefs found in %s", briefsDir)
	}

	// Filter out high-frequency keywords that appear in too many briefs
	// (these chain everything into one mega-cluster via single-linkage)
	FilterCommonKeywords(briefs, MaxDocumentFrequency)

	threads, err := LoadThreads(threadsDir)
	if err != nil {
		return nil, fmt.Errorf("loading threads: %w", err)
	}

	clusters := ClusterBriefs(briefs)
	unclustered := UnclusteredBriefs(briefs, clusters)

	var digestClusters []*DigestCluster
	for _, c := range clusters {
		dc := &DigestCluster{Cluster: c}
		if len(threads) > 0 {
			dc.ThreadMatches = MatchClusterToThreads(c, threads)
		}
		digestClusters = append(digestClusters, dc)
	}

	// Harvest tensions from unclustered briefs
	var orphanTensions []TensionEntry
	for _, b := range unclustered {
		if b.Tension != "" {
			orphanTensions = append(orphanTensions, TensionEntry{
				BriefID: b.ID,
				Text:    b.Tension,
			})
		}
	}

	return &Digest{
		Date:            time.Now(),
		BriefsComposed:  len(briefs),
		ClustersFound:   len(digestClusters),
		Clusters:        digestClusters,
		Unclustered:     unclustered,
		TensionOrphans:  orphanTensions,
		EpistemicStatus: DefaultEpistemicStatus,
	}, nil
}

// WriteDigest writes a digest to the digests directory as markdown.
func WriteDigest(digest *Digest, digestsDir string) (string, error) {
	if err := os.MkdirAll(digestsDir, 0755); err != nil {
		return "", fmt.Errorf("creating digests directory: %w", err)
	}

	filename := fmt.Sprintf("%s-digest.md", digest.Date.Format("2006-01-02"))
	path := filepath.Join(digestsDir, filename)

	content := renderDigest(digest)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing digest: %w", err)
	}

	return path, nil
}

func renderDigest(d *Digest) string {
	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("date: %s\n", d.Date.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("briefs_composed: %d\n", d.BriefsComposed))
	sb.WriteString(fmt.Sprintf("clusters_found: %d\n", d.ClustersFound))
	sb.WriteString("epistemic_status: unverified-clustering\n")
	sb.WriteString("---\n\n")

	// Epistemic Status
	sb.WriteString("## Epistemic Status\n\n")
	sb.WriteString(d.EpistemicStatus)
	sb.WriteString("\n\n")

	// Clusters
	for i, dc := range d.Clusters {
		sb.WriteString(fmt.Sprintf("## Cluster %d: %s\n\n", i+1, dc.Name))

		// Brief IDs
		ids := make([]string, len(dc.Briefs))
		for j, b := range dc.Briefs {
			ids[j] = b.ID
		}
		sb.WriteString(fmt.Sprintf("**Briefs:** %s\n\n", strings.Join(ids, ", ")))

		// Clustering rationale
		sb.WriteString(fmt.Sprintf("**Why clustered:** %s\n\n", dc.Rationale))

		// Thread connections
		if len(dc.ThreadMatches) > 0 {
			for _, tm := range dc.ThreadMatches {
				sb.WriteString(fmt.Sprintf("**Thread connection:** May relate to thread \"%s\" (shared: %s)\n\n",
					tm.Thread.Title, strings.Join(tm.SharedKeywords, ", ")))
			}
		}

		// Harvested tensions from cluster members
		sb.WriteString("### Harvested tensions\n\n")
		hasTensions := false
		for _, b := range dc.Briefs {
			if b.Tension != "" {
				sb.WriteString(fmt.Sprintf("- (%s) %s\n", b.ID, b.Tension))
				hasTensions = true
			}
		}
		if !hasTensions {
			sb.WriteString("- (none)\n")
		}
		sb.WriteString("\n")

		// Draft thread proposal
		if len(dc.ThreadMatches) > 0 {
			sb.WriteString("### Draft thread proposal\n\n")
			best := dc.ThreadMatches[0]
			sb.WriteString(fmt.Sprintf("**Proposal:** Append to thread \"%s\" — %d briefs share patterns around: %s\n\n",
				best.Thread.Title, len(dc.Briefs), strings.Join(dc.SharedKeywords, ", ")))
		} else if len(dc.SharedKeywords) > 0 {
			sb.WriteString("### Draft thread proposal\n\n")
			sb.WriteString(fmt.Sprintf("**Proposal:** New thread about \"%s\" — %d briefs converge on this topic with no existing thread match.\n\n",
				dc.Name, len(dc.Briefs)))
		}

		sb.WriteString("---\n\n")
	}

	// Unclustered briefs
	if len(d.Unclustered) > 0 {
		sb.WriteString("## Unclustered Briefs\n\n")
		for _, b := range d.Unclustered {
			summary := b.Frame
			if len(summary) > 120 {
				summary = summary[:120] + "..."
			}
			sb.WriteString(fmt.Sprintf("- **%s** — %s\n", b.ID, summary))
		}
		sb.WriteString("\n")
	}

	// Tension orphans
	if len(d.TensionOrphans) > 0 {
		sb.WriteString("## Tension Orphans\n\n")
		sb.WriteString("Tensions from unclustered briefs — open questions with no structural home:\n\n")
		for _, t := range d.TensionOrphans {
			sb.WriteString(fmt.Sprintf("- (%s) %s\n", t.BriefID, t.Text))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
