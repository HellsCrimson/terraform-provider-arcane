// Command statelint is a static analyzer that detects the "stale-state Update"
// bug class in this Terraform provider.
//
// Background: most Update handlers seed their result from the PRIOR state
// (`req.State.Get(ctx, &state)`) and then copy only *some* attributes back
// before `resp.State.Set(ctx, &state)`. Any user-settable attribute the handler
// forgets to copy keeps its old value, producing:
//
//	Error: Provider produced inconsistent result after apply
//	... produced an unexpected new value: .<attr>: was X, but now Y.
//
// See STALE_STATE_UPDATE_FINDINGS.md for the full write-up.
//
// This tool parses every resource_*.go, links each Schema to its model struct
// and Update handler, and reports settable attributes that are not safely
// persisted. It intentionally over-reports nothing it can prove safe:
//
//   - Computed / RequiresReplace attributes are excluded (not user-settable in
//     an in-place Update, or masked by a default).
//   - Handlers that rebuild wholesale (`state = plan`) are immune.
//   - Required attributes only need to be assigned on *some* path (their guards
//     `if !plan.X.IsNull()` always fire, since Required values are never null).
//   - Optional attributes must be assigned on *every* path (an `if` without a
//     matching `else` misses the clear-to-null transition).
//
// It also flags the adjacent case where Update unconditionally errors
// ("immutable") while settable attributes lack RequiresReplace, so editing them
// routes to the erroring Update instead of forcing a clean replace.
//
// Usage:
//
//	go run ./tools/statelint [dir-or-file ...]   # defaults to ./internal/provider
//
// Exit code is 1 when any findings are reported (suitable for CI).
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// attrInfo captures the schema properties of a single top-level attribute.
type attrInfo struct {
	optional        bool
	computed        bool
	required        bool
	requiresReplace bool
}

// settable reports whether a change to this attribute routes through Update in
// place (and therefore must be persisted back into state).
func (a attrInfo) settable() bool {
	if a.computed || a.requiresReplace {
		return false
	}
	return a.optional || a.required
}

// updateInfo captures what an Update handler does with its state variable.
type updateInfo struct {
	found     bool
	stateVar  string          // variable passed to resp.State.Set(ctx, &X)
	modelType string          // Go type of stateVar
	rebuild   bool            // `stateVar = plan` — all fields persisted
	immutable bool            // no State.Set + an AddError (network/volume shape)
	topFields map[string]bool // Go fields assigned on every path
	anyFields map[string]bool // Go fields assigned on some path
	pos       token.Position
}

// resourceInfo bundles everything the analysis needs for one receiver type.
type resourceInfo struct {
	recv   string
	schema map[string]attrInfo // tfsdk name -> info
	update updateInfo
	models map[string]map[string]string // modelType -> (goField -> tfsdk name)
	file   string
}

type finding struct {
	file, attr, goField, kind, fix string
	schema                         string
}

func main() {
	targets := os.Args[1:]
	if len(targets) == 0 {
		targets = []string{"internal/provider"}
	}

	files, err := goFiles(targets)
	if err != nil {
		fmt.Fprintln(os.Stderr, "statelint:", err)
		os.Exit(2)
	}

	fset := token.NewFileSet()
	var findings []finding
	for _, f := range files {
		ff, err := analyzeFile(fset, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "statelint: %s: %v\n", f, err)
			os.Exit(2)
		}
		findings = append(findings, ff...)
	}

	report(findings)
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func goFiles(targets []string) ([]string, error) {
	var out []string
	for _, t := range targets {
		info, err := os.Stat(t)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			out = append(out, t)
			continue
		}
		entries, err := os.ReadDir(t)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			out = append(out, filepath.Join(t, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

func analyzeFile(fset *token.FileSet, path string) ([]finding, error) {
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	// Collect models (structs with tfsdk tags), schemas, and Update handlers.
	models := map[string]map[string]string{} // modelType -> goField -> tfsdk
	schemas := map[string]map[string]attrInfo{}
	updates := map[string]updateInfo{}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if fields := modelFields(st); len(fields) > 0 {
					models[ts.Name.Name] = fields
				}
			}
		case *ast.FuncDecl:
			recv := receiverType(d)
			if recv == "" {
				continue
			}
			switch d.Name.Name {
			case "Schema":
				if attrs := parseSchema(d); attrs != nil {
					schemas[recv] = attrs
				}
			case "Update":
				updates[recv] = parseUpdate(fset, d)
			}
		}
	}

	var findings []finding
	for recv, schema := range schemas {
		upd, ok := updates[recv]
		if !ok || !upd.found {
			continue
		}
		// Immutable handlers have no State.Set, so the model type can't be read
		// from the state var. Resolve it by matching schema attrs to a model.
		if upd.modelType == "" {
			upd.modelType = resolveModel(schema, models)
		}
		ri := resourceInfo{recv: recv, schema: schema, update: upd, models: models, file: path}
		findings = append(findings, ri.analyze()...)
	}
	return findings, nil
}

// modelFields returns goField -> tfsdk name for a struct that uses tfsdk tags.
func modelFields(st *ast.StructType) map[string]string {
	out := map[string]string{}
	for _, f := range st.Fields.List {
		if f.Tag == nil || len(f.Names) == 0 {
			continue
		}
		tag, err := strconv.Unquote(f.Tag.Value)
		if err != nil {
			continue
		}
		name := reflect.StructTag(tag).Get("tfsdk")
		if name == "" {
			continue
		}
		out[f.Names[0].Name] = name
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// resolveModel picks the model struct whose tfsdk-name set covers all schema
// attributes (each resource's model mirrors its schema). Used when the Update
// handler doesn't reveal the model type (immutable, State.Set-less handlers).
func resolveModel(schema map[string]attrInfo, models map[string]map[string]string) string {
	best, bestScore := "", -1
	for name, fields := range models {
		tfsdk := map[string]bool{}
		for _, tn := range fields {
			tfsdk[tn] = true
		}
		covered := 0
		for attr := range schema {
			if tfsdk[attr] {
				covered++
			}
		}
		if covered == len(schema) && len(fields) > bestScore {
			best, bestScore = name, len(fields)
		}
	}
	return best
}

func receiverType(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	t := d.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// parseSchema extracts the top-level Attributes map from a Schema method.
func parseSchema(d *ast.FuncDecl) map[string]attrInfo {
	var attrsLit *ast.CompositeLit
	ast.Inspect(d, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Attributes" {
			if cl, ok := kv.Value.(*ast.CompositeLit); ok && attrsLit == nil {
				attrsLit = cl
			}
		}
		return true
	})
	if attrsLit == nil {
		return nil
	}

	out := map[string]attrInfo{}
	for _, elt := range attrsLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := stringLit(kv.Key)
		if !ok {
			continue
		}
		out[key] = parseAttr(kv.Value)
	}
	return out
}

func parseAttr(v ast.Expr) attrInfo {
	var info attrInfo
	cl, ok := v.(*ast.CompositeLit)
	if !ok {
		return info
	}
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		id, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch id.Name {
		case "Optional":
			info.optional = isTrue(kv.Value)
		case "Computed":
			info.computed = isTrue(kv.Value)
		case "Required":
			info.required = isTrue(kv.Value)
		}
	}
	// RequiresReplace may appear nested inside PlanModifiers.
	ast.Inspect(v, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "RequiresReplace" {
			info.requiresReplace = true
		}
		return true
	})
	return info
}

// parseUpdate walks an Update handler and records how it treats its state var.
func parseUpdate(fset *token.FileSet, d *ast.FuncDecl) updateInfo {
	info := updateInfo{found: true, pos: fset.Position(d.Pos()), topFields: map[string]bool{}, anyFields: map[string]bool{}}
	if d.Body == nil {
		return info
	}

	// Variable declarations: varName -> type.
	varTypes := map[string]string{}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		gd, ok := n.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			return true
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || vs.Type == nil {
				continue
			}
			if id, ok := vs.Type.(*ast.Ident); ok {
				for _, name := range vs.Names {
					varTypes[name.Name] = id.Name
				}
			}
		}
		return true
	})

	// State variable = argument of resp.State.Set(ctx, &X). The prior-state
	// variable = argument of req.State.Get(ctx, &X).
	hasStateSet := false
	hasAddError := false
	priorStateVar := ""
	ast.Inspect(d.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "AddError" {
			hasAddError = true
		}
		if v, ok := stateAddrArg(sel, call, "Set"); ok && isStateSelector(sel.X) {
			hasStateSet = true
			info.stateVar = v
		}
		if v, ok := stateAddrArg(sel, call, "Get"); ok && isStateSelector(sel.X) {
			priorStateVar = v
		}
		return true
	})
	info.immutable = !hasStateSet && hasAddError
	if info.stateVar == "" {
		return info
	}
	info.modelType = varTypes[info.stateVar]

	// The stale-state bug requires the Set variable to BE the prior-state
	// variable (carried forward). A different variable is freshly built, and a
	// whole-variable reassignment (`stateVar = plan`, `newState := build(...)`)
	// discards any prior state — both are immune.
	if info.stateVar != priorStateVar || wholeReassigned(d.Body, info.stateVar) {
		info.rebuild = true
	}

	// Field assignment coverage.
	info.topFields = assignedAllPaths(d.Body, info.stateVar)
	info.anyFields = assignedAny(d.Body, info.stateVar)
	return info
}

// stateAddrArg returns the identifier X from a call `<recv>.<method>(_, &X)`
// when sel.Sel.Name == method and the second arg is a `&ident`.
func stateAddrArg(sel *ast.SelectorExpr, call *ast.CallExpr, method string) (string, bool) {
	if sel.Sel.Name != method || len(call.Args) < 2 {
		return "", false
	}
	u, ok := call.Args[1].(*ast.UnaryExpr)
	if !ok || u.Op != token.AND {
		return "", false
	}
	if id, ok := u.X.(*ast.Ident); ok {
		return id.Name, true
	}
	return "", false
}

// wholeReassigned reports whether stateVar appears as a whole-identifier LHS of
// any assignment (i.e. the entire struct is replaced, discarding prior state).
func wholeReassigned(body *ast.BlockStmt, stateVar string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range as.Lhs {
			if id, ok := lhs.(*ast.Ident); ok && id.Name == stateVar {
				found = true
			}
		}
		return true
	})
	return found
}

// isStateSelector reports whether expr denotes `resp.State` (i.e. the Set call
// writes provider state rather than plan or private data).
func isStateSelector(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "State"
}

// assignedAny returns every state field assigned on any path within node.
func assignedAny(node ast.Node, stateVar string) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(node, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range as.Lhs {
			if f := stateField(lhs, stateVar); f != "" {
				out[f] = true
			}
		}
		return true
	})
	return out
}

// assignedAllPaths returns the state fields assigned on EVERY path through stmt.
func assignedAllPaths(stmt ast.Stmt, stateVar string) map[string]bool {
	out := map[string]bool{}
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, child := range s.List {
			for f := range assignedAllPaths(child, stateVar) {
				out[f] = true
			}
		}
	case *ast.AssignStmt:
		for _, lhs := range s.Lhs {
			if f := stateField(lhs, stateVar); f != "" {
				out[f] = true
			}
		}
	case *ast.IfStmt:
		if s.Else == nil {
			return out // then-only branch guarantees nothing
		}
		thenF := assignedAllPaths(s.Body, stateVar)
		elseF := assignedAllPaths(s.Else, stateVar)
		for f := range thenF {
			if elseF[f] {
				out[f] = true
			}
		}
	}
	return out
}

// stateField returns the Go field name if expr is `stateVar.Field`, else "".
func stateField(expr ast.Expr, stateVar string) string {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Name != stateVar {
		return ""
	}
	return sel.Sel.Name
}

func (ri resourceInfo) analyze() []finding {
	upd := ri.update
	fieldToTfsdk := ri.models[upd.modelType]
	if fieldToTfsdk == nil {
		return nil
	}
	// Reverse: tfsdk name -> Go field.
	tfsdkToField := map[string]string{}
	for gf, tn := range fieldToTfsdk {
		tfsdkToField[tn] = gf
	}

	var findings []finding
	names := make([]string, 0, len(ri.schema))
	for n := range ri.schema {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, attr := range names {
		info := ri.schema[attr]
		if !info.settable() {
			continue
		}
		goField, ok := tfsdkToField[attr]
		if !ok {
			continue // schema attr without a model field; skip
		}

		if upd.immutable {
			findings = append(findings, finding{
				file: ri.file, attr: attr, goField: goField, schema: schemaKind(info),
				kind: "immutable-update",
				fix:  "add stringplanmodifier.RequiresReplace() so edits force a clean replace instead of the erroring Update",
			})
			continue
		}
		if upd.rebuild || upd.topFields[goField] {
			continue // safely persisted
		}
		if info.required {
			if upd.anyFields[goField] {
				continue // guard always fires for Required attrs
			}
			findings = append(findings, finding{
				file: ri.file, attr: attr, goField: goField, schema: schemaKind(info),
				kind: "not-persisted",
				fix:  fmt.Sprintf("add `state.%s = plan.%s` before resp.State.Set", goField, goField),
			})
			continue
		}
		// Optional from here.
		if upd.anyFields[goField] {
			findings = append(findings, finding{
				file: ri.file, attr: attr, goField: goField, schema: schemaKind(info),
				kind: "conditional-clear-to-null",
				fix:  fmt.Sprintf("assign `state.%s = plan.%s` unconditionally (add the else branch) so clearing it also persists", goField, goField),
			})
			continue
		}
		findings = append(findings, finding{
			file: ri.file, attr: attr, goField: goField, schema: schemaKind(info),
			kind: "not-persisted",
			fix:  fmt.Sprintf("add `state.%s = plan.%s` before resp.State.Set", goField, goField),
		})
	}
	// Stable order within a file.
	sort.Slice(findings, func(i, j int) bool { return findings[i].attr < findings[j].attr })
	return findings
}

func schemaKind(info attrInfo) string {
	switch {
	case info.required:
		return "Required"
	case info.optional && info.computed:
		return "Optional+Computed"
	case info.optional:
		return "Optional"
	default:
		return "?"
	}
}

func report(findings []finding) {
	if len(findings) == 0 {
		fmt.Println("statelint: no stale-state Update findings")
		return
	}
	byFile := map[string][]finding{}
	var order []string
	for _, f := range findings {
		if _, ok := byFile[f.file]; !ok {
			order = append(order, f.file)
		}
		byFile[f.file] = append(byFile[f.file], f)
	}
	sort.Strings(order)

	total := 0
	for _, file := range order {
		fmt.Printf("\n%s\n", file)
		for _, f := range byFile[file] {
			total++
			fmt.Printf("  [%s] %-24s (%s, field %s)\n", f.kind, f.attr, f.schema, f.goField)
			fmt.Printf("      fix: %s\n", f.fix)
		}
	}
	fmt.Printf("\nstatelint: %d finding(s) across %d file(s)\n", total, len(order))
}

func stringLit(e ast.Expr) (string, bool) {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(bl.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

func isTrue(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "true"
}
