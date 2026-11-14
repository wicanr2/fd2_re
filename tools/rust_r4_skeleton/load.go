package main

// 載入 remake/cmd/fd2 的正式程式（不含 _test.go），以 go list -export 產生的匯出資料
// 檢查型別。build tag 照 go list 的 GoFiles（linux／amd64）；被排除的檔另列在 PORTING.json。

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type goFile struct {
	Name  string
	Path  string
	AST   *ast.File
	Lines int
	Src   []byte
}

type listPkg struct {
	ImportPath     string
	Dir            string
	Export         string
	GoFiles        []string
	IgnoredGoFiles []string
}

type pkgData struct {
	fset    *token.FileSet
	files   []*goFile
	byPath  map[string]*goFile
	ignored []string
	pkg     *types.Package
	info    *types.Info
}

const mainImportPath = "github.com/wicanr2/fd2_re/remake/cmd/fd2"

// loadPackage 讀 go list -export -deps -json 的輸出（多個 JSON 物件串接）。
func loadPackage(depsPath string) *pkgData {
	f, err := os.Open(depsPath)
	must(err)
	defer f.Close()
	dec := json.NewDecoder(f)
	exports := map[string]string{}
	var main *listPkg
	for {
		var p listPkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else {
			must(err)
		}
		if p.Export != "" {
			exports[p.ImportPath] = p.Export
		}
		if p.ImportPath == mainImportPath {
			pp := p
			main = &pp
		}
	}
	if main == nil {
		panic("go list 輸出沒有 " + mainImportPath)
	}
	fset := token.NewFileSet()
	pd := &pkgData{fset: fset, byPath: map[string]*goFile{}}
	var astFiles []*ast.File
	names := append([]string{}, main.GoFiles...)
	sort.Strings(names)
	for _, n := range names {
		p := filepath.Join(main.Dir, n)
		src, err := os.ReadFile(p)
		must(err)
		af, err := parser.ParseFile(fset, p, src, parser.ParseComments)
		must(err)
		gf := &goFile{Name: n, Path: p, AST: af, Src: src}
		gf.Lines = strings.Count(string(src), "\n")
		if len(src) > 0 && src[len(src)-1] != '\n' {
			gf.Lines++
		}
		pd.files = append(pd.files, gf)
		pd.byPath[p] = gf
		astFiles = append(astFiles, af)
	}
	for _, n := range main.IgnoredGoFiles {
		if !strings.HasSuffix(n, "_test.go") {
			pd.ignored = append(pd.ignored, n)
		}
	}
	sort.Strings(pd.ignored)
	pd.info = &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Implicits:  map[ast.Node]types.Object{},
		Scopes:     map[ast.Node]*types.Scope{},
	}
	lookup := func(path string) (io.ReadCloser, error) {
		e, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("沒有 %s 的匯出資料", path)
		}
		return os.Open(e)
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "gc", lookup)}
	pkg, err := conf.Check(mainImportPath, fset, astFiles, pd.info)
	must(err)
	pd.pkg = pkg
	return pd
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (pd *pkgData) fileOf(pos token.Pos) *goFile {
	return pd.byPath[pd.fset.Position(pos).Filename]
}

func (pd *pkgData) line(pos token.Pos) int { return pd.fset.Position(pos).Line }

type coverageFunc struct {
	Func  string  `json:"func"`
	Class string  `json:"class"`
	Cover float64 `json:"cover"`
}

type coverageDoc struct {
	GoSourceCommit string                    `json:"go_source_commit"`
	Files          map[string][]coverageFunc `json:"files"`
}

func loadCoverage(p string) *coverageDoc {
	data, err := os.ReadFile(p)
	must(err)
	var c coverageDoc
	must(json.Unmarshal(data, &c))
	return &c
}
