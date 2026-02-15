package tree

import "time"

// NodeType represents the type of knowledge artifact
type NodeType string

const (
	NodeTypeInvestigation NodeType = "investigation"
	NodeTypeDecision      NodeType = "decision"
	NodeTypeModel         NodeType = "model"
	NodeTypeProbe         NodeType = "probe"
	NodeTypeGuide         NodeType = "guide"
	NodeTypeIssue         NodeType = "issue"
	NodeTypeCluster       NodeType = "cluster"
	NodeTypePostMortem    NodeType = "postmortem"
	NodeTypeHandoff       NodeType = "handoff"
)

// NodeStatus represents the status of a node (for investigations and issues)
type NodeStatus string

const (
	StatusComplete   NodeStatus = "complete"
	StatusTriage     NodeStatus = "triage"
	StatusInProgress NodeStatus = "in_progress"
	StatusClosed     NodeStatus = "closed"
	StatusOpen       NodeStatus = "open"
)

// KnowledgeNode represents a node in the knowledge tree
type KnowledgeNode struct {
	ID       string                 // Unique identifier (file path or beads ID)
	Type     NodeType               // Type of node
	Title    string                 // Display title
	Path     string                 // File path (for file-based artifacts) or beads ID (for issues)
	Status   NodeStatus             // Status (if applicable)
	Date     time.Time              // Creation or update date
	Children []*KnowledgeNode       // Child nodes
	Metadata map[string]interface{} // Additional metadata
}

// Relationship represents a relationship between artifacts
type Relationship struct {
	From         string // Source file path
	To           string // Target file path
	RelationType string // Type of relationship (synthesizes, references, extends, etc.)
	Verified     bool   // Whether the relationship was verified
}

// HealthSmellType represents different types of health smells
type HealthSmellType string

const (
	SmellNeedsSynthesis HealthSmellType = "needs_synthesis" // 15+ investigations without decision/model
	SmellNotActedOn     HealthSmellType = "not_acted_on"    // Decision without spawned issues
	SmellUntestedModel  HealthSmellType = "untested_model"  // Model without probes
)

// HealthSmell represents a health smell detected in a cluster
type HealthSmell struct {
	Type        HealthSmellType // Type of smell
	Description string          // Human-readable description
	Count       int             // Supporting count (e.g., number of investigations)
}

// Cluster represents a group of related artifacts
type Cluster struct {
	Name   string           // Cluster name
	Nodes  []*KnowledgeNode // Nodes in this cluster
	Smells []HealthSmell    // Health smells detected in this cluster
}

// TreeOptions represents options for rendering the tree
type TreeOptions struct {
	ClusterFilter string // Filter to specific cluster
	Depth         int    // Maximum depth to render (0 = unlimited)
	Format        string // Output format (text or json)
	WorkView      bool   // Use work view instead of knowledge view
	SmellsOnly    bool   // Filter to only clusters with health smells
	Compact       bool   // Use compact format (minimal output for hook injection)
}
