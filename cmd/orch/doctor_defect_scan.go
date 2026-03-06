package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DefectFinding represents a single defect class instance found in the codebase.
type DefectFinding struct {
	Class       int    `json:"class"`
	ClassName   string `json:"class_name"`
	File        string `json:"file"`
	Function    string `json:"function"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // "high", "medium", "low"
}

// DefectScanReport contains the results of a defect class scan.
type DefectScanReport struct {
	Findings     []DefectFinding `json:"findings"`
	FilesScanned int             `json:"files_scanned"`
	Class2Count  int             `json:"class_2_count"`
	Class5Count  int             `json:"class_5_count"`
}

// funcInfo tracks what backend calls a function makes.
type funcInfo struct {
	name      string
	file      string
	startLine int
	// Class 2: backend call tracking
	hasOpenCodeCall bool
	hasTmuxCall     bool
	hasDualPattern  bool // uses DiscoverLiveAgents, queryTrackedAgents, etc.
	// Class 5: authority signal tracking
	hasPhaseCheck     bool
	hasSynthesisCheck bool
	hasSessionStatus  bool
	hasTmuxLiveness   bool
	hasBeadsStatus    bool
	// Context
	isAgentRelated bool // function touches agent discovery/status/lifecycle
}

// Known patterns for detection
var (
	// Class 2: OpenCode-only patterns
	openCodePatterns = []string{
		"client.ListSessions",
		"client.GetSession",
		"client.GetMessages",
		"client.GetSessionStatus",
		"client.CreateSession",
		"client.SendMessage",
		"ListSessions(",
		"GetSessionStatusByIDs(",
	}

	// Class 2: Tmux-only patterns
	tmuxPatterns = []string{
		"tmux.ListWindows",
		"tmux.FindWindow",
		"tmux.SessionExists",
		"tmux.WindowExists",
		"tmux.SendKeys",
		"tmux.CapturePane",
		"tmux.KillWindow",
		"tmux.CreateWindow",
	}

	// Class 2: Dual-backend patterns (exempt from single-backend warnings)
	dualBackendPatterns = []string{
		"DiscoverLiveAgents",
		"queryTrackedAgents",
		"checkTmuxWindowAlive",
		"CountActiveTmuxAgents",
		"joinWithReasonCodes",
		// pkg/discovery canonical interface
		"discovery.QueryTrackedAgents",
		"discovery.JoinWithReasonCodes",
		"discovery.ListTrackedIssues",
		"discovery.CheckTmuxWindowAlive",
		"discovery.FilterActiveIssues",
	}

	// Class 5: Phase comment signals
	phaseSignalPatterns = []string{
		"ParsePhaseFromComments",
		"IsPhaseComplete",
		"GetPhaseStatus",
		"HasBeadsComment",
		`Phase: Complete`,
		`"Phase:"`,
		"phaseComplete",
	}

	// Class 5: Synthesis artifact signals
	synthesisSignalPatterns = []string{
		"SYNTHESIS.md",
		"checkWorkspaceSynthesis",
		"hasSynthesis",
	}

	// Class 5: Session status signals
	sessionStatusPatterns = []string{
		"sessionStatus",
		"SessionDead",
		`"idle"`,
		`"busy"`,
		`"active"`,
		"getSessionStatus",
	}

	// Class 5: Tmux liveness signals
	tmuxLivenessPatterns = []string{
		"checkTmuxWindowAlive",
		"WindowExists",
		"tmux_window_alive",
	}

	// Class 5: Beads issue status signals
	beadsStatusPatterns = []string{
		"IssueClosed",
		"issueClosed",
		`issue.Status`,
		"CloseIssue",
		"ForceCloseIssue",
	}

	// Agent-related keywords (function must touch these to be relevant)
	agentKeywords = []string{
		"agent", "Agent",
		"session", "Session",
		"status", "Status",
		"spawn", "Spawn",
		"complete", "Complete",
		"liveness", "Liveness",
		"active", "Active",
		"discovery", "Discovery",
		"cleanup", "Cleanup",
		"clean", "Clean",
		"abandon", "Abandon",
	}

	// Functions known to correctly handle both backends (allowlist)
	class2Allowlist = map[string]bool{
		"DiscoverLiveAgents":    true,
		"queryTrackedAgents":     true,
		"joinWithReasonCodes":    true,
		"CountActiveTmuxAgents":  true,
		"checkTmuxWindowAlive":   true,
		// pkg/discovery canonical interface (backend-aware by design)
		"QueryTrackedAgents":         true,
		"JoinWithReasonCodes":        true,
		"ListTrackedIssues":          true,
		"CheckTmuxWindowAlive":       true,
		"FilterActiveIssues":         true,
		"ExtractSessionIDs":          true,
		"ExtractLatestPhases":        true,
		"LatestPhaseFromComments":    true,
		"LatestPhaseWithTimestamp":   true,
		"UnknownLiveness":            true,
		"LookupManifestsAcrossProjects": true,
		"determineAgentStatus":   true,
		"runQuestion":            true, // has fallback chain
		"runTail":                true, // has fallback chain
		"checkStalledSessions":   true, // intentionally OpenCode-only (OpenCode sessions)
		"runSessionsCrossReference": true,
		// Backend-specific helpers (intentionally single-backend by design)
		"verifyOpencodeDeliverables":  true, // backend-specific verification
		"verifyTmuxDeliverables":      true, // backend-specific verification
		"SpawnClaude":                 true, // Claude CLI spawn (tmux by design)
		"AbandonClaude":              true, // Claude CLI cleanup (tmux by design)
		"startHeadlessSession":       true, // OpenCode headless (by design)
		"runSpawnInline":             true, // OpenCode inline (by design)
		"EnsureOpenCodeRunning":      true, // OpenCode health check
		"readFromOpenCodeMetadata":   true, // OpenCode-specific metadata
		"handleCompletion":           true, // OpenCode completion handler
		"checkOpenCode":              true, // doctor: OpenCode health check
		"startOpenCode":              true, // doctor: start OpenCode
		"cleanupTmuxWindow":         true, // intentionally tmux-only cleanup
		"GenerateServerContext":      true, // spawn context uses tmux for server management
		"checkOpenCodeSession":       true, // reconcile: intentionally checks one backend
		"checkTmuxWindow":           true, // reconcile: intentionally checks one backend
		// Server management (tmux-only by design — servers run in tmux)
		"runServersStart":  true,
		"runServersStop":   true,
		"runServersAttach": true,
		// Session-specific operations (operate on known session type)
		"sendViaOpenCodeAPI": true,
		"handleSessionMessages": true,
		"handleSessions":    true,
		// Lifecycle operations that correctly route by spawn mode upstream
		"Complete":      true,
		"ForceComplete": true,
		"Abandon":       true,
		"ForceAbandon":  true,
		"DetectOrphans": true,
		// OpenCode client methods (low-level, not agent discovery)
		"ListSessions":         true,
		"GetSessionStatusByIDs": true,
		"GetSessionStatusByID": true,
	}

	// Functions known to correctly handle authority signals (allowlist)
	class5Allowlist = map[string]bool{
		"determineAgentStatus": true, // canonical priority cascade
		"ListUnverifiedWork":   true, // canonical verification source
		"parseFunctions":       true, // this file's own scanner (matches signal patterns as strings)
	}

	funcDeclRegex = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
)

// runDefectScan performs a static analysis scan for Class 2 and Class 5 defect patterns.
func runDefectScan() error {
	fmt.Println("orch doctor --defect-scan")
	fmt.Println("Scanning for defect class patterns (Class 2: Multi-Backend Blindness, Class 5: Contradictory Authority Signals)...")
	fmt.Println()

	report, err := scanForDefects(".")
	if err != nil {
		return fmt.Errorf("defect scan error: %w", err)
	}

	// Print results
	fmt.Printf("Files scanned: %d\n", report.FilesScanned)
	fmt.Printf("Class 2 findings: %d\n", report.Class2Count)
	fmt.Printf("Class 5 findings: %d\n", report.Class5Count)
	fmt.Println()

	if len(report.Findings) == 0 {
		fmt.Println("✓ No defect class patterns detected")
		return nil
	}

	// Group by class
	class2 := make([]DefectFinding, 0)
	class5 := make([]DefectFinding, 0)
	for _, f := range report.Findings {
		switch f.Class {
		case 2:
			class2 = append(class2, f)
		case 5:
			class5 = append(class5, f)
		}
	}

	if len(class2) > 0 {
		fmt.Println("── Class 2: Multi-Backend Blindness ──")
		fmt.Println()
		for _, f := range class2 {
			icon := severityIcon(f.Severity)
			fmt.Printf("  %s %s:%d  %s()\n", icon, f.File, f.Line, f.Function)
			fmt.Printf("      %s\n", f.Description)
		}
		fmt.Println()
	}

	if len(class5) > 0 {
		fmt.Println("── Class 5: Contradictory Authority Signals ──")
		fmt.Println()
		for _, f := range class5 {
			icon := severityIcon(f.Severity)
			fmt.Printf("  %s %s:%d  %s()\n", icon, f.File, f.Line, f.Function)
			fmt.Printf("      %s\n", f.Description)
		}
		fmt.Println()
	}

	fmt.Println("Prevention:")
	if len(class2) > 0 {
		fmt.Println("  Class 2: Use pkg/discovery.QueryTrackedAgents() or DiscoverLiveAgents() for agent discovery")
	}
	if len(class5) > 0 {
		fmt.Println("  Class 5: Use determineAgentStatus() as canonical status derivation")
	}

	return nil
}

func severityIcon(severity string) string {
	switch severity {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	default:
		return "🔵"
	}
}

// scanForDefects scans Go files in the given directory for Class 2 and Class 5 patterns.
func scanForDefects(rootDir string) (*DefectScanReport, error) {
	report := &DefectScanReport{
		Findings: make([]DefectFinding, 0),
	}

	// Scan directories
	scanDirs := []string{
		filepath.Join(rootDir, "cmd", "orch"),
		filepath.Join(rootDir, "pkg"),
	}

	for _, dir := range scanDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			// Skip test files
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			report.FilesScanned++
			findings, err := scanFile(path, rootDir)
			if err != nil {
				return nil // skip file errors
			}
			report.Findings = append(report.Findings, findings...)
			return nil
		})
		if err != nil {
			// Directory might not exist, skip
			continue
		}
	}

	// Sort findings by class, then severity, then file
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Class != report.Findings[j].Class {
			return report.Findings[i].Class < report.Findings[j].Class
		}
		if report.Findings[i].Severity != report.Findings[j].Severity {
			return severityRank(report.Findings[i].Severity) < severityRank(report.Findings[j].Severity)
		}
		return report.Findings[i].File < report.Findings[j].File
	})

	// Count by class
	for _, f := range report.Findings {
		switch f.Class {
		case 2:
			report.Class2Count++
		case 5:
			report.Class5Count++
		}
	}

	return report, nil
}

func severityRank(s string) int {
	switch s {
	case "high":
		return 0
	case "medium":
		return 1
	default:
		return 2
	}
}

// scanFile scans a single Go file for defect patterns, returning findings.
func scanFile(path string, rootDir string) ([]DefectFinding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Parse file into functions
	funcs := parseFunctions(f)

	// Compute relative path for display
	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		relPath = path
	}

	var findings []DefectFinding

	for _, fn := range funcs {
		if !fn.isAgentRelated {
			continue
		}

		// Class 2 checks
		findings = append(findings, checkClass2(fn, relPath)...)

		// Class 5 checks
		findings = append(findings, checkClass5(fn, relPath)...)
	}

	return findings, nil
}

// parseFunctions parses a Go source file and extracts function-level pattern info.
func parseFunctions(f *os.File) []funcInfo {
	var funcs []funcInfo
	var current *funcInfo
	braceDepth := 0
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check for function declaration
		if matches := funcDeclRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			// Save previous function
			if current != nil {
				funcs = append(funcs, *current)
			}
			current = &funcInfo{
				name:      matches[1],
				startLine: lineNum,
			}
			braceDepth = 0
		}

		if current == nil {
			continue
		}

		// Track brace depth
		braceDepth += strings.Count(line, "{") - strings.Count(line, "}")

		// Check for agent-related keywords
		if !current.isAgentRelated {
			for _, kw := range agentKeywords {
				if strings.Contains(line, kw) {
					current.isAgentRelated = true
					break
				}
			}
		}

		// Check for OpenCode patterns
		for _, p := range openCodePatterns {
			if strings.Contains(line, p) {
				current.hasOpenCodeCall = true
				break
			}
		}

		// Check for tmux patterns
		for _, p := range tmuxPatterns {
			if strings.Contains(line, p) {
				current.hasTmuxCall = true
				break
			}
		}

		// Check for dual-backend patterns
		for _, p := range dualBackendPatterns {
			if strings.Contains(line, p) {
				current.hasDualPattern = true
				break
			}
		}

		// Check for Class 5 authority signals
		for _, p := range phaseSignalPatterns {
			if strings.Contains(line, p) {
				current.hasPhaseCheck = true
				break
			}
		}
		for _, p := range synthesisSignalPatterns {
			if strings.Contains(line, p) {
				current.hasSynthesisCheck = true
				break
			}
		}
		for _, p := range sessionStatusPatterns {
			if strings.Contains(line, p) {
				current.hasSessionStatus = true
				break
			}
		}
		for _, p := range tmuxLivenessPatterns {
			if strings.Contains(line, p) {
				current.hasTmuxLiveness = true
				break
			}
		}
		for _, p := range beadsStatusPatterns {
			if strings.Contains(line, p) {
				current.hasBeadsStatus = true
				break
			}
		}

		// End of function (brace depth returns to 0 after seeing opening brace)
		if braceDepth <= 0 && lineNum > current.startLine {
			funcs = append(funcs, *current)
			current = nil
		}
	}

	// Don't forget the last function
	if current != nil {
		funcs = append(funcs, *current)
	}

	return funcs
}

// checkClass2 checks a function for Multi-Backend Blindness patterns.
func checkClass2(fn funcInfo, relPath string) []DefectFinding {
	var findings []DefectFinding

	// Skip if function is in the allowlist
	if class2Allowlist[fn.name] {
		return nil
	}

	// Skip if function uses a dual-backend pattern (delegating correctly)
	if fn.hasDualPattern {
		return nil
	}

	// Flag: OpenCode call without tmux check
	if fn.hasOpenCodeCall && !fn.hasTmuxCall {
		severity := "medium"
		// Higher severity for functions that clearly do agent discovery/counting
		if containsAny(fn.name, "Count", "List", "Active", "Discovery", "Status") {
			severity = "high"
		}
		findings = append(findings, DefectFinding{
			Class:       2,
			ClassName:   "Multi-Backend Blindness",
			File:        relPath,
			Function:    fn.name,
			Line:        fn.startLine,
			Description: "OpenCode query without tmux check — Claude CLI agents invisible",
			Severity:    severity,
		})
	}

	// Flag: Tmux call without OpenCode check (for agent lifecycle operations)
	if fn.hasTmuxCall && !fn.hasOpenCodeCall {
		severity := "medium"
		if containsAny(fn.name, "Count", "List", "Active", "Discovery", "Clean") {
			severity = "high"
		}
		findings = append(findings, DefectFinding{
			Class:       2,
			ClassName:   "Multi-Backend Blindness",
			File:        relPath,
			Function:    fn.name,
			Line:        fn.startLine,
			Description: "Tmux check without OpenCode query — headless agents invisible",
			Severity:    severity,
		})
	}

	return findings
}

// checkClass5 checks a function for Contradictory Authority Signals patterns.
func checkClass5(fn funcInfo, relPath string) []DefectFinding {
	var findings []DefectFinding

	// Skip if function is in the allowlist
	if class5Allowlist[fn.name] {
		return nil
	}

	// Count how many distinct authority signals this function reads
	signalCount := 0
	var signals []string

	if fn.hasPhaseCheck {
		signalCount++
		signals = append(signals, "phase-comments")
	}
	if fn.hasSynthesisCheck {
		signalCount++
		signals = append(signals, "synthesis-artifact")
	}
	if fn.hasSessionStatus {
		signalCount++
		signals = append(signals, "session-status")
	}
	if fn.hasTmuxLiveness {
		signalCount++
		signals = append(signals, "tmux-liveness")
	}
	if fn.hasBeadsStatus {
		signalCount++
		signals = append(signals, "beads-issue-status")
	}

	// Flag: Function reads 3+ authority signals (high risk of contradictory logic)
	if signalCount >= 3 {
		severity := "high"
		findings = append(findings, DefectFinding{
			Class:       5,
			ClassName:   "Contradictory Authority Signals",
			File:        relPath,
			Function:    fn.name,
			Line:        fn.startLine,
			Description: fmt.Sprintf("Reads %d authority signals (%s) — verify precedence is explicit", signalCount, strings.Join(signals, ", ")),
			Severity:    severity,
		})
	} else if signalCount == 2 {
		// 2 signals is lower risk but still worth noting
		severity := "low"
		if containsAny(fn.name, "Status", "Complete", "Determine", "Derive") {
			severity = "medium"
		}
		findings = append(findings, DefectFinding{
			Class:       5,
			ClassName:   "Contradictory Authority Signals",
			File:        relPath,
			Function:    fn.name,
			Line:        fn.startLine,
			Description: fmt.Sprintf("Reads 2 authority signals (%s) — ensure consistent precedence", strings.Join(signals, ", ")),
			Severity:    severity,
		})
	}

	return findings
}

// containsAny returns true if s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
