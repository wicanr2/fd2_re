// 使用 Go AST 核對正式戰鬥快取 owner 與受控消費端清冊的覆蓋。
// 執行入口：Docker 內 go run tools/audit_fd2_battle_owners.go -root /repo
// -observed <原始觀測.json> -physical-normal <反擊核對.json> -out <全新輸出.json>。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	root := flag.String("root", "/repo", "儲存庫根目錄")
	observed := flag.String("observed", "", "受控原始觀測")
	physical := flag.String("physical-normal", "", "正常反擊核對收據")
	out := flag.String("out", "", "全新輸出")
	flag.Parse()
	must := func(ok bool, message string) {
		if !ok {
			panic(message)
		}
	}
	read := func(path string) []byte {
		data, err := os.ReadFile(path)
		must(err == nil, fmt.Sprint(err))
		return data
	}
	sha := func(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
	must(*out != "" && *observed != "" && *physical != "", "缺必要參數")
	_, err := os.Stat(*out)
	must(os.IsNotExist(err), "輸出必須是全新路徑")
	var observation struct {
		Status string                             `json:"status"`
		Cases  []struct{ ID, Constructor string } `json:"cases"`
	}
	must(json.Unmarshal(read(*observed), &observation) == nil && observation.Status == "CONTROLLED-CONSUMER-OBSERVATIONS", "觀測格式不同")
	var normal struct {
		Counter struct {
			Status  string
			Matches []json.RawMessage
		} `json:"normal_counter"`
	}
	must(json.Unmarshal(read(*physical), &normal) == nil && normal.Counter.Status == "PASS-NORMAL-COUNTER-SCOPED" && len(normal.Counter.Matches) > 0, "缺正常物理反擊證據")
	set := token.NewFileSet()
	functions := map[string]*ast.FuncDecl{}
	files := map[string]string{}
	paths, err := filepath.Glob(filepath.Join(*root, "remake/cmd/fd2/*.go"))
	must(err == nil, "列舉來源失敗")
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		must(err == nil, fmt.Sprint(err))
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			kind, ok := star.X.(*ast.Ident)
			if !ok || kind.Name != "Game" {
				continue
			}
			must(functions[fn.Name.Name] == nil, "Game方法重複："+fn.Name.Name)
			functions[fn.Name.Name], files[fn.Name.Name] = fn, path
		}
	}
	expression := func(node ast.Node) string {
		var b bytes.Buffer
		must(format.Node(&b, set, node) == nil, "AST格式失敗")
		return b.String()
	}
	registry := map[string]bool{}
	sync := functions["syncModernHDBattleOwner"]
	must(sync != nil, "缺正式owner同步入口")
	ast.Inspect(sync.Body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, left := range assign.Lhs {
			name, ok := left.(*ast.Ident)
			if !ok || name.Name != "owner" || i >= len(assign.Rhs) {
				continue
			}
			right := expression(assign.Rhs[i])
			if strings.HasPrefix(right, "g.") {
				registry[strings.TrimPrefix(right, "g.")] = true
			}
		}
		return true
	})
	must(registry["atk.nativeScene"], "物理owner未登錄")
	covered := map[string]bool{"atk.nativeScene": true}
	var constructors []map[string]any
	sourceHashes := map[string]string{}
	remember := func(path string) {
		rel, err := filepath.Rel(*root, path)
		must(err == nil, "來源路徑失敗")
		sourceHashes[filepath.ToSlash(rel)] = sha(read(path))
	}
	remember(files["syncModernHDBattleOwner"])
	for _, item := range observation.Cases {
		if strings.HasPrefix(item.ID, "physical/") {
			continue
		}
		fn := functions[item.Constructor]
		must(fn != nil, "找不到正式建構器："+item.Constructor)
		published := map[string]bool{}
		visited := map[string]bool{}
		var inspectConstructor func(string)
		inspectConstructor = func(name string) {
			if visited[name] || functions[name] == nil {
				return
			}
			visited[name] = true
			remember(files[name])
			ast.Inspect(functions[name].Body, func(node ast.Node) bool {
				// Some public entry points return a cursor-aware constructor.
				// Follow that explicit delegation; this remains a structural audit.
				if ret, ok := node.(*ast.ReturnStmt); ok {
					for _, result := range ret.Results {
						call, ok := result.(*ast.CallExpr)
						if !ok {
							continue
						}
						selector, ok := call.Fun.(*ast.SelectorExpr)
						if ok && expression(selector.X) == "g" {
							inspectConstructor(selector.Sel.Name)
						}
					}
				}
				assign, ok := node.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i, left := range assign.Lhs {
					name := strings.TrimPrefix(expression(left), "g.")
					if !registry[name] || i >= len(assign.Rhs) || expression(assign.Rhs[i]) == "nil" {
						continue
					}
					published[name], covered[name] = true, true
				}
				return true
			})
		}
		inspectConstructor(item.Constructor)
		must(len(published) > 0, "建構器沒有發布已登錄owner："+item.Constructor)
		owners := make([]string, 0, len(published))
		for owner := range published {
			owners = append(owners, owner)
		}
		sort.Strings(owners)
		var inspected []string
		for name := range visited {
			inspected = append(inspected, name)
		}
		sort.Strings(inspected)
		constructors = append(constructors, map[string]any{"case": item.ID, "constructor": item.Constructor, "owners": owners, "inspected_functions": inspected, "line": set.Position(fn.Pos()).Line})
		remember(files[item.Constructor])
	}
	var names, missing []string
	for owner := range registry {
		names = append(names, owner)
		if !covered[owner] {
			missing = append(missing, owner)
		}
	}
	sort.Strings(names)
	sort.Strings(missing)
	must(len(missing) == 0, "未覆蓋owner："+strings.Join(missing, ","))
	report := map[string]any{"status": "PASS-CACHED-BATTLE-OWNER-COVERAGE", "owner_registry": names, "uncovered_owners": missing, "constructors": constructors,
		"source_files_sha256": sourceHashes, "observed_sha256": sha(read(*observed)), "physical_normal_sha256": sha(read(*physical)),
		"scope": "Go AST的正式戰鬥快取owner發布覆蓋；物理由正常反擊收據佐證。其他地圖owner、全部參數組合與原版parity另驗，不從快取清單推定可達性。"}
	data, err := json.MarshalIndent(report, "", "  ")
	must(err == nil, "JSON失敗")
	must(os.WriteFile(*out, append(data, '\n'), 0644) == nil, "寫入失敗")
	fmt.Printf("PASS cached owners=%d constructors=%d\n", len(names), len(constructors))
}
