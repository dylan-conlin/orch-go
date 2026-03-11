// Package dupdetect provides AST-based function duplication detection.
//
// It parses Go source files, extracts structural fingerprints from function
// bodies (normalizing identifier names), and compares them to find similar
// function pairs. This is Harness Layer 2 — detecting code clones that
// indicate extraction opportunities.
package dupdetect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FuncInfo describes a parsed function for comparison.
type FuncInfo struct {
	File        string
	Name        string
	Lines       int
	StartLine   int
	Fingerprint []string // normalized AST node sequence
}

// DupPair represents a pair of functions with high structural similarity.
type DupPair struct {
	FuncA      FuncInfo
	FuncB      FuncInfo
	Similarity float64
}

// Detector configures and runs duplication detection.
type Detector struct {
	MinBodyLines int      // skip functions smaller than this (default 10)
	Threshold    float64  // similarity threshold 0.0-1.0 (default 0.80)
	Allowlist    []string // function name patterns — pairs where both match same pattern are suppressed
}

// NewDetector returns a Detector with sensible defaults.
func NewDetector() *Detector {
	return &Detector{
		MinBodyLines: 10,
		Threshold:    0.80,
	}
}

// ScanDir parses all non-test .go files in dir and returns duplicate pairs.
func (d *Detector) ScanDir(dir string) ([]DupPair, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var allFuncs []FuncInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		filePath := filepath.Join(dir, name)
		src, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		funcs, err := d.ParseSource(name, string(src))
		if err != nil {
			continue // skip unparseable files
		}
		allFuncs = append(allFuncs, funcs...)
	}

	return d.FindDuplicates(allFuncs), nil
}

// ParseSource extracts FuncInfo from Go source code.
func (d *Detector) ParseSource(filename, src string) ([]FuncInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return nil, err
	}

	var funcs []FuncInfo
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		startLine := fset.Position(fn.Body.Lbrace).Line
		endLine := fset.Position(fn.Body.Rbrace).Line
		bodyLines := endLine - startLine + 1

		if bodyLines < d.MinBodyLines {
			continue
		}

		name := funcName(fn)
		fp := fingerprint(fn.Body)

		funcs = append(funcs, FuncInfo{
			File:        filename,
			Name:        name,
			Lines:       bodyLines,
			StartLine:   fset.Position(fn.Pos()).Line,
			Fingerprint: fp,
		})
	}
	return funcs, nil
}

// FindDuplicates compares all function pairs and returns those above threshold.
func (d *Detector) FindDuplicates(funcs []FuncInfo) []DupPair {
	var pairs []DupPair

	for i := 0; i < len(funcs); i++ {
		for j := i + 1; j < len(funcs); j++ {
			if !canMeetThreshold(funcs[i].Fingerprint, funcs[j].Fingerprint, d.Threshold) {
				continue
			}
			sim := similarity(funcs[i].Fingerprint, funcs[j].Fingerprint)
			if sim >= d.Threshold {
				if len(d.Allowlist) > 0 && isAllowlisted(funcs[i].Name, funcs[j].Name, d.Allowlist) {
					continue
				}
				pairs = append(pairs, DupPair{
					FuncA:      funcs[i],
					FuncB:      funcs[j],
					Similarity: sim,
				})
			}
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Similarity > pairs[j].Similarity
	})
	return pairs
}

// FindDuplicatesAgainst compares each function in "modified" against all
// functions in "corpus" (plus modified-vs-modified). This is O(M×N) where
// M=len(modified) instead of O(N²) for the full corpus.
// Functions in modified should NOT also appear in corpus — deduplicate before calling.
func (d *Detector) FindDuplicatesAgainst(modified, corpus []FuncInfo) []DupPair {
	var pairs []DupPair

	// modified vs corpus (M × N)
	for i := range modified {
		for j := range corpus {
			if !canMeetThreshold(modified[i].Fingerprint, corpus[j].Fingerprint, d.Threshold) {
				continue
			}
			sim := similarity(modified[i].Fingerprint, corpus[j].Fingerprint)
			if sim >= d.Threshold {
				if len(d.Allowlist) > 0 && isAllowlisted(modified[i].Name, corpus[j].Name, d.Allowlist) {
					continue
				}
				pairs = append(pairs, DupPair{
					FuncA:      modified[i],
					FuncB:      corpus[j],
					Similarity: sim,
				})
			}
		}
	}

	// modified vs modified (M × M, typically tiny)
	for i := 0; i < len(modified); i++ {
		for j := i + 1; j < len(modified); j++ {
			if !canMeetThreshold(modified[i].Fingerprint, modified[j].Fingerprint, d.Threshold) {
				continue
			}
			sim := similarity(modified[i].Fingerprint, modified[j].Fingerprint)
			if sim >= d.Threshold {
				if len(d.Allowlist) > 0 && isAllowlisted(modified[i].Name, modified[j].Name, d.Allowlist) {
					continue
				}
				pairs = append(pairs, DupPair{
					FuncA:      modified[i],
					FuncB:      modified[j],
					Similarity: sim,
				})
			}
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Similarity > pairs[j].Similarity
	})
	return pairs
}

// fingerprint walks an AST block and produces a sequence of normalized
// node type tokens. Identifiers are replaced with positional placeholders
// so that renamed-variable clones are detected as identical.
func fingerprint(body *ast.BlockStmt) []string {
	var tokens []string
	idMap := map[string]string{} // original name -> normalized placeholder
	nextID := 0

	normalizeIdent := func(name string) string {
		if n, ok := idMap[name]; ok {
			return n
		}
		n := fmt.Sprintf("$%d", nextID)
		nextID++
		idMap[name] = n
		return n
	}

	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		switch x := n.(type) {
		case *ast.Ident:
			tokens = append(tokens, "IDENT:"+normalizeIdent(x.Name))
		case *ast.BasicLit:
			tokens = append(tokens, "LIT:"+x.Kind.String()+":"+x.Value)
		case *ast.AssignStmt:
			tokens = append(tokens, "ASSIGN:"+x.Tok.String())
		case *ast.BinaryExpr:
			tokens = append(tokens, "BINARY:"+x.Op.String())
		case *ast.UnaryExpr:
			tokens = append(tokens, "UNARY:"+x.Op.String())
		case *ast.CallExpr:
			tokens = append(tokens, "CALL")
		case *ast.ReturnStmt:
			tokens = append(tokens, "RETURN")
		case *ast.IfStmt:
			tokens = append(tokens, "IF")
		case *ast.ForStmt:
			tokens = append(tokens, "FOR")
		case *ast.RangeStmt:
			tokens = append(tokens, "RANGE")
		case *ast.SwitchStmt:
			tokens = append(tokens, "SWITCH")
		case *ast.TypeSwitchStmt:
			tokens = append(tokens, "TYPESWITCH")
		case *ast.SelectStmt:
			tokens = append(tokens, "SELECT")
		case *ast.DeferStmt:
			tokens = append(tokens, "DEFER")
		case *ast.GoStmt:
			tokens = append(tokens, "GO")
		case *ast.SendStmt:
			tokens = append(tokens, "SEND")
		case *ast.BranchStmt:
			tokens = append(tokens, "BRANCH:"+x.Tok.String())
		case *ast.IncDecStmt:
			tokens = append(tokens, "INCDEC:"+x.Tok.String())
		case *ast.IndexExpr:
			tokens = append(tokens, "INDEX")
		case *ast.SliceExpr:
			tokens = append(tokens, "SLICE")
		case *ast.TypeAssertExpr:
			tokens = append(tokens, "TYPEASSERT")
		case *ast.KeyValueExpr:
			tokens = append(tokens, "KV")
		case *ast.CompositeLit:
			tokens = append(tokens, "COMPOSITE")
		case *ast.FuncLit:
			tokens = append(tokens, "FUNCLIT")
		case *ast.SelectorExpr:
			tokens = append(tokens, "SEL")
		case *ast.StarExpr:
			tokens = append(tokens, "STAR")
		case *ast.BlockStmt:
			tokens = append(tokens, "BLOCK")
		case *ast.ExprStmt:
			tokens = append(tokens, "EXPR")
		case *ast.DeclStmt:
			tokens = append(tokens, "DECL")
		}
		return true
	})
	return tokens
}

// canMeetThreshold is a cheap O(1) pre-filter. The LCS-based similarity
// can never exceed min(len(a),len(b))/max(len(a),len(b)), so if that
// ratio is already below threshold we skip the expensive LCS computation.
func canMeetThreshold(a, b []string, threshold float64) bool {
	la, lb := len(a), len(b)
	if la == 0 && lb == 0 {
		return true
	}
	if la == 0 || lb == 0 {
		return false
	}
	minLen, maxLen := la, lb
	if minLen > maxLen {
		minLen, maxLen = maxLen, minLen
	}
	return float64(minLen)/float64(maxLen) >= threshold
}

// similarity computes the similarity between two fingerprints using
// longest common subsequence (LCS) ratio.
func similarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	lcs := lcsLength(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	return float64(lcs) / float64(maxLen)
}

// lcsLength computes the length of the longest common subsequence.
// Uses O(min(m,n)) space.
func lcsLength(a, b []string) int {
	if len(a) < len(b) {
		a, b = b, a
	}
	// b is the shorter sequence
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1] + 1
			} else {
				curr[j] = prev[j]
				if curr[j-1] > curr[j] {
					curr[j] = curr[j-1]
				}
			}
		}
		prev, curr = curr, prev
		for k := range curr {
			curr[k] = 0
		}
	}
	return prev[len(b)]
}

func funcName(fn *ast.FuncDecl) string {
	name := fn.Name.Name
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
			if ident, ok := star.X.(*ast.Ident); ok {
				return fmt.Sprintf("(%s).%s", ident.Name, name)
			}
		} else if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
			return fmt.Sprintf("(%s).%s", ident.Name, name)
		}
	}
	return name
}
