package boundedlifetime

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	linterName = "boundedlifetime"

	goroutineMessage = "goroutine must accept a context.Context parameter"
	execMessage      = "use exec.CommandContext with context.WithTimeout instead of exec.Command"
	ctxTimeoutMsg    = "exec.CommandContext must receive a context created by context.WithTimeout"

	cacheBoundAndEvictionMsg = "map cache field %q requires max-size bound and eviction logic"
	cacheBoundMsg            = "map cache field %q requires max-size bound"
	cacheEvictionMsg         = "map cache field %q requires eviction logic that deletes entries"
)

func newAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: linterName,
		Doc:  "flags goroutines, subprocesses, and cache fields without bounded lifetimes",
		Run:  run,
	}
}

func run(pass *analysis.Pass) (any, error) {
	timeoutVars := collectTimeoutContextVars(pass)

	for _, file := range pass.Files {
		if isTestFile(pass, file) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.GoStmt:
				reportGoroutineWithoutContext(pass, node)
			case *ast.CallExpr:
				reportUnboundedExec(pass, node, timeoutVars)
			case *ast.TypeSpec:
				reportCacheStructWithoutBounds(pass, node)
			}

			return true
		})
	}

	return nil, nil
}

func isTestFile(pass *analysis.Pass, file *ast.File) bool {
	filename := pass.Fset.Position(file.Pos()).Filename
	return strings.HasSuffix(filepath.Base(filename), "_test.go")
}

func reportGoroutineWithoutContext(pass *analysis.Pass, stmt *ast.GoStmt) {
	if stmt.Call == nil {
		return
	}

	funcLit, ok := stmt.Call.Fun.(*ast.FuncLit)
	if !ok {
		return
	}

	if funcLitHasContextParam(pass, funcLit) {
		return
	}

	pass.Reportf(stmt.Pos(), goroutineMessage)
}

func funcLitHasContextParam(pass *analysis.Pass, lit *ast.FuncLit) bool {
	if lit.Type.Params == nil {
		return false
	}

	for _, field := range lit.Type.Params.List {
		if isContextType(pass.TypesInfo.TypeOf(field.Type)) {
			return true
		}
	}

	return false
}

func reportUnboundedExec(pass *analysis.Pass, call *ast.CallExpr, timeoutVars map[*types.Var]struct{}) {
	if isPackageCall(pass, call, "os/exec", "Command") {
		pass.Reportf(call.Pos(), execMessage)
		return
	}

	if !isPackageCall(pass, call, "os/exec", "CommandContext") {
		return
	}

	if len(call.Args) == 0 || !hasTimeoutContextArg(pass, call.Args[0], timeoutVars) {
		pass.Reportf(call.Pos(), ctxTimeoutMsg)
	}
}

func hasTimeoutContextArg(pass *analysis.Pass, arg ast.Expr, timeoutVars map[*types.Var]struct{}) bool {
	if isContextWithTimeoutCall(pass, arg) {
		return true
	}

	id, ok := arg.(*ast.Ident)
	if !ok {
		return false
	}

	obj, ok := pass.TypesInfo.Uses[id].(*types.Var)
	if !ok {
		return false
	}

	_, found := timeoutVars[obj]
	return found
}

func collectTimeoutContextVars(pass *analysis.Pass) map[*types.Var]struct{} {
	vars := make(map[*types.Var]struct{})

	for _, file := range pass.Files {
		if isTestFile(pass, file) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				markTimeoutVarsFromAssign(pass, node, vars)
			case *ast.ValueSpec:
				markTimeoutVarsFromValueSpec(pass, node, vars)
			}

			return true
		})
	}

	return vars
}

func markTimeoutVarsFromAssign(pass *analysis.Pass, stmt *ast.AssignStmt, vars map[*types.Var]struct{}) {
	if len(stmt.Rhs) == 1 && isContextWithTimeoutCall(pass, stmt.Rhs[0]) {
		if len(stmt.Lhs) > 0 {
			markVarFromExpr(pass, stmt.Lhs[0], vars)
		}
		return
	}

	n := min(len(stmt.Lhs), len(stmt.Rhs))
	for i := 0; i < n; i++ {
		if isContextWithTimeoutCall(pass, stmt.Rhs[i]) {
			markVarFromExpr(pass, stmt.Lhs[i], vars)
		}
	}
}

func markTimeoutVarsFromValueSpec(pass *analysis.Pass, spec *ast.ValueSpec, vars map[*types.Var]struct{}) {
	if len(spec.Values) == 1 && isContextWithTimeoutCall(pass, spec.Values[0]) {
		if len(spec.Names) > 0 {
			markVarFromIdent(pass, spec.Names[0], vars)
		}
		return
	}

	n := min(len(spec.Names), len(spec.Values))
	for i := 0; i < n; i++ {
		if isContextWithTimeoutCall(pass, spec.Values[i]) {
			markVarFromIdent(pass, spec.Names[i], vars)
		}
	}
}

func markVarFromExpr(pass *analysis.Pass, expr ast.Expr, vars map[*types.Var]struct{}) {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return
	}

	markVarFromIdent(pass, id, vars)
}

func markVarFromIdent(pass *analysis.Pass, id *ast.Ident, vars map[*types.Var]struct{}) {
	if id == nil {
		return
	}

	if def, ok := pass.TypesInfo.Defs[id].(*types.Var); ok {
		vars[def] = struct{}{}
		return
	}

	if use, ok := pass.TypesInfo.Uses[id].(*types.Var); ok {
		vars[use] = struct{}{}
	}
}

func reportCacheStructWithoutBounds(pass *analysis.Pass, spec *ast.TypeSpec) {
	structType, ok := spec.Type.(*ast.StructType)
	if !ok {
		return
	}

	requireAllMapFields := strings.Contains(strings.ToLower(spec.Name.Name), "cache")
	cacheFields := cacheMapFields(pass, structType, requireAllMapFields)
	if len(cacheFields) == 0 {
		return
	}

	named, ok := namedTypeForSpec(pass, spec)
	if !ok {
		return
	}

	hasMaxBound := hasMaxSizeBound(pass, structType)
	hasEviction := hasMapEvictionMethod(pass, named, cacheFields)
	if hasMaxBound && hasEviction {
		return
	}

	for _, field := range cacheFields {
		for _, name := range field.Names {
			switch {
			case !hasMaxBound && !hasEviction:
				pass.Reportf(name.Pos(), cacheBoundAndEvictionMsg, name.Name)
			case !hasMaxBound:
				pass.Reportf(name.Pos(), cacheBoundMsg, name.Name)
			case !hasEviction:
				pass.Reportf(name.Pos(), cacheEvictionMsg, name.Name)
			}
		}
	}
}

func hasMaxSizeBound(pass *analysis.Pass, st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		if !isIntegerType(pass.TypesInfo.TypeOf(field.Type)) {
			continue
		}

		for _, name := range field.Names {
			if isMaxSizeFieldName(name.Name) {
				return true
			}
		}
	}

	return false
}

func isIntegerType(t types.Type) bool {
	if t == nil {
		return false
	}

	basic, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok {
		return false
	}

	info := basic.Info()
	return info&types.IsInteger != 0
}

func isMaxSizeFieldName(name string) bool {
	lower := strings.ToLower(name)
	if !strings.Contains(lower, "max") {
		return false
	}

	for _, token := range []string{"size", "entry", "entries", "item", "items", "capacity", "cap", "limit"} {
		if strings.Contains(lower, token) {
			return true
		}
	}

	return false
}

func hasMapEvictionMethod(pass *analysis.Pass, named *types.Named, cacheFields []*ast.Field) bool {
	cacheFieldNames := make(map[string]struct{})
	for _, field := range cacheFields {
		for _, name := range field.Names {
			cacheFieldNames[name.Name] = struct{}{}
		}
	}

	if len(cacheFieldNames) == 0 {
		return false
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}

			if !methodBelongsToNamedType(pass, fn, named) {
				continue
			}

			if methodDeletesCacheField(pass, fn, named, cacheFieldNames) {
				return true
			}
		}
	}

	return false
}

func methodBelongsToNamedType(pass *analysis.Pass, fn *ast.FuncDecl, named *types.Named) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}

	recvType := pass.TypesInfo.TypeOf(fn.Recv.List[0].Type)
	if recvType == nil {
		return false
	}

	recvNamed, ok := namedTypeFromType(recvType)
	if !ok {
		return false
	}

	return sameNamedType(recvNamed, named)
}

func methodDeletesCacheField(pass *analysis.Pass, fn *ast.FuncDecl, named *types.Named, fields map[string]struct{}) bool {
	deleted := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		fun, ok := call.Fun.(*ast.Ident)
		if !ok || fun.Name != "delete" || len(call.Args) < 1 {
			return true
		}

		sel, ok := call.Args[0].(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if _, ok := fields[sel.Sel.Name]; !ok {
			return true
		}

		if !exprIsNamedReceiver(pass, sel.X, named) {
			return true
		}

		deleted = true
		return false
	})

	return deleted
}

func exprIsNamedReceiver(pass *analysis.Pass, expr ast.Expr, named *types.Named) bool {
	exprType := pass.TypesInfo.TypeOf(expr)
	if exprType == nil {
		return false
	}

	exprNamed, ok := namedTypeFromType(exprType)
	if !ok {
		return false
	}

	return sameNamedType(exprNamed, named)
}

func namedTypeFromType(t types.Type) (*types.Named, bool) {
	t = types.Unalias(t)
	if ptr, ok := t.(*types.Pointer); ok {
		t = types.Unalias(ptr.Elem())
	}

	named, ok := t.(*types.Named)
	return named, ok
}

func sameNamedType(a, b *types.Named) bool {
	if a == nil || b == nil {
		return false
	}

	aObj := a.Obj()
	bObj := b.Obj()
	if aObj == nil || bObj == nil {
		return false
	}

	if aObj.Name() != bObj.Name() {
		return false
	}

	aPkg := aObj.Pkg()
	bPkg := bObj.Pkg()
	if aPkg == nil || bPkg == nil {
		return false
	}

	return aPkg.Path() == bPkg.Path()
}

func cacheMapFields(pass *analysis.Pass, st *ast.StructType, requireAllMapFields bool) []*ast.Field {
	if st.Fields == nil {
		return nil
	}

	var out []*ast.Field
	for _, field := range st.Fields.List {
		if !isMapType(pass.TypesInfo.TypeOf(field.Type)) {
			continue
		}

		if requireAllMapFields || fieldHasCacheName(field) {
			out = append(out, field)
		}
	}

	return out
}

func fieldHasCacheName(field *ast.Field) bool {
	for _, name := range field.Names {
		if strings.Contains(strings.ToLower(name.Name), "cache") {
			return true
		}
	}

	return false
}

func namedTypeForSpec(pass *analysis.Pass, spec *ast.TypeSpec) (*types.Named, bool) {
	obj, ok := pass.TypesInfo.Defs[spec.Name].(*types.TypeName)
	if !ok {
		return nil, false
	}

	named, ok := obj.Type().(*types.Named)
	return named, ok
}

func isMapType(t types.Type) bool {
	if t == nil {
		return false
	}

	_, ok := types.Unalias(t).Underlying().(*types.Map)
	return ok
}

func isContextType(t types.Type) bool {
	if t == nil {
		return false
	}

	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

func isContextWithTimeoutCall(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	return isPackageCall(pass, call, "context", "WithTimeout")
}

func isPackageCall(pass *analysis.Pass, call *ast.CallExpr, pkgPath, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	obj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || obj.Pkg() == nil {
		return false
	}

	return obj.Name() == name && obj.Pkg().Path() == pkgPath
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
