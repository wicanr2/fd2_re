//go:build ignore

// Run in the maintained Go Docker image. This audits current Go references;
// it is not a typed call graph, original oracle, or player reachability proof.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type function struct {
	Symbol    string   `json:"symbol"`
	File      string   `json:"file"`
	Line      int      `json:"line"`
	End       int      `json:"end_line"`
	Refs      []string `json:"references"`
	Callers   []string `json:"referenced_by"`
	Reachable bool     `json:"reachable_from_game_roots"`
	body      *ast.BlockStmt
}

type job struct {
	Kind      string   `json:"kind"`
	Function  string   `json:"function"`
	File      string   `json:"file"`
	Line      int      `json:"line"`
	Fields    []string `json:"fields"`
	Reachable bool     `json:"reachable_from_game_roots"`
}

func receiver(e ast.Expr) string {
	if p, ok := e.(*ast.StarExpr); ok {
		return receiver(p.X)
	}
	if p, ok := e.(*ast.Ident); ok {
		return p.Name
	}
	return ""
}

func main() {
	repo := flag.String("repo", "", "explicit repository root")
	out := flag.String("out", "", "explicit existing owned output directory and JSON filename")
	flag.Parse()
	if _, err := os.Stat("/.dockerenv"); err != nil {
		panic("Docker required")
	}
	if *repo == "" || *out == "" {
		panic("--repo and --out required")
	}
	dir := filepath.Join(*repo, "remake/cmd/fd2")
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		panic(err)
	}
	sort.Strings(names)
	set := token.NewFileSet()
	functions := map[string]*function{}
	games, free := map[string]string{}, map[string]string{}
	digest := sha256.New()
	files := 0
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		match, err := build.Default.MatchFile(dir, filepath.Base(name))
		if err != nil {
			panic(err)
		}
		if !match {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			panic(err)
		}
		rel, err := filepath.Rel(*repo, name)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(digest, "%s\x00", filepath.ToSlash(rel))
		digest.Write(raw)
		digest.Write([]byte{0})
		files++
		file, err := parser.ParseFile(set, name, raw, 0)
		if err != nil {
			panic(err)
		}
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || f.Body == nil {
				continue
			}
			owner := ""
			if f.Recv != nil && len(f.Recv.List) == 1 {
				owner = receiver(f.Recv.List[0].Type)
			}
			symbol := f.Name.Name
			if owner != "" {
				symbol = owner + "." + symbol
			}
			if functions[symbol] != nil {
				panic("duplicate active symbol: " + symbol)
			}
			functions[symbol] = &function{Symbol: symbol, File: filepath.ToSlash(rel), Line: set.Position(f.Pos()).Line, End: set.Position(f.End()).Line, body: f.Body}
			if owner == "Game" {
				games[f.Name.Name] = symbol
			} else if owner == "" {
				free[f.Name.Name] = symbol
			}
		}
	}
	jobs := []job{}
	for _, f := range functions {
		refs := map[string]bool{}
		ast.Inspect(f.body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if symbol := games[x.Sel.Name]; symbol != "" {
					refs[symbol] = true
				}
			case *ast.Ident:
				if symbol := free[x.Name]; symbol != "" {
					refs[symbol] = true
				}
			case *ast.CompositeLit:
				kind := receiver(x.Type)
				if kind != "nativeClassUIJob" && kind != "nativeChurchUIJob" {
					break
				}
				j := job{Kind: kind, Function: f.Symbol, File: f.File, Line: set.Position(x.Pos()).Line}
				for _, el := range x.Elts {
					if kv, ok := el.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok {
							j.Fields = append(j.Fields, key.Name)
						}
					}
				}
				sort.Strings(j.Fields)
				jobs = append(jobs, j)
			}
			return true
		})
		for ref := range refs {
			f.Refs = append(f.Refs, ref)
		}
		sort.Strings(f.Refs)
	}
	for _, f := range functions {
		for _, ref := range f.Refs {
			functions[ref].Callers = append(functions[ref].Callers, f.Symbol)
		}
	}
	roots := []string{"Game.Update", "Game.Draw", "Game.enterNode"}
	queue := append([]string(nil), roots...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		f := functions[name]
		if f == nil {
			panic("missing game root: " + name)
		}
		if f.Reachable {
			continue
		}
		f.Reachable = true
		queue = append(queue, f.Refs...)
	}
	ordered := []*function{}
	unused := []*function{}
	presenters := []*function{}
	for _, f := range functions {
		sort.Strings(f.Callers)
		ordered = append(ordered, f)
		if len(f.Callers) == 0 && strings.HasPrefix(f.Symbol, "Game.") && (strings.Contains(f.Symbol, "NativeChurch") || strings.Contains(f.Symbol, "NativeShop") || strings.Contains(f.Symbol, "NativeHotel") || strings.Contains(f.Symbol, "NativeClass")) {
			unused = append(unused, f)
		}
		for _, ref := range f.Refs {
			if ref == "Game.presentNativeClassFrame" || ref == "Game.presentNativeClassFrameWithPalette" {
				presenters = append(presenters, f)
				break
			}
		}
	}
	less := func(a, b *function) bool { return a.Symbol < b.Symbol }
	sort.Slice(ordered, func(i, j int) bool { return less(ordered[i], ordered[j]) })
	sort.Slice(unused, func(i, j int) bool { return less(unused[i], unused[j]) })
	sort.Slice(presenters, func(i, j int) bool { return less(presenters[i], presenters[j]) })
	for i := range jobs {
		jobs[i].Reachable = functions[jobs[i].Function].Reachable
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].File == jobs[j].File {
			return jobs[i].Line < jobs[j].Line
		}
		return jobs[i].File < jobs[j].File
	})
	report := map[string]any{"schema": "fd2-hd1-interface-source-audit-v1", "scope": "current build-matched production Go source; conservative symbol references include method values and callback bodies; name matching can over-approximate reachability; no original oracle or normal-player proof",
		"go_version": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH, "source_files": files, "source_tree_sha256": hex.EncodeToString(digest.Sum(nil)), "roots": roots, "functions": ordered, "job_constructors": jobs, "no_runtime_reference_facility_helpers": unused, "native_presenter_functions": presenters}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	if _, err = os.Stat(filepath.Dir(*out)); err != nil {
		panic(err)
	}
	if err = os.WriteFile(*out, append(raw, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("source_files=%d functions=%d jobs=%d no_reference_facility_helpers=%d native_presenters=%d\n", files, len(functions), len(jobs), len(unused), len(presenters))
}
