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
	"strconv"
	"strings"
)

func main() {
	root := flag.String("root", "/repo", "儲存庫根目錄")
	observed := flag.String("observed", "", "受控原始觀測")
	physical := flag.String("physical-normal", "", "正常反擊核對收據")
	out := flag.String("out", "", "全新輸出")
	routesOnly := flag.Bool("routes-only", false, "只核對36個正式指令的選單閘門與confirm派送，不提升為動畫完成")
	flag.Parse()
	if *routesOnly {
		auditCommandRoutes(*root, *out)
		return
	}
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

// auditCommandRoutes 與快取owner覆蓋分開。動畫建構器存在，仍可能被選單擋住。
// 這個模式只報AST與正式資料的事實，不以靜態呼叫證明GPU、正常取得或返回。
func auditCommandRoutes(root, out string) {
	must := func(ok bool, message string) {
		if !ok {
			panic(message)
		}
	}
	must(out != "", "缺輸出路徑")
	_, err := os.Stat(out)
	must(os.IsNotExist(err), "輸出必須是全新路徑")
	hashes := map[string]string{}
	read := func(relative string) []byte {
		data, err := os.ReadFile(filepath.Join(root, relative))
		must(err == nil, fmt.Sprint(err))
		hashes[relative] = fmt.Sprintf("%x", sha256.Sum256(data))
		return data
	}
	set := token.NewFileSet()
	functions := map[string]*ast.FuncDecl{}
	locations := map[string]string{}
	for _, relative := range []string{"remake/cmd/fd2/main.go", "remake/cmd/fd2/native_ai_execute.go"} {
		file, err := parser.ParseFile(set, relative, read(relative), 0)
		must(err == nil, fmt.Sprint(err))
		for _, declaration := range file.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok {
				must(functions[fn.Name.Name] == nil, "函式重複："+fn.Name.Name)
				functions[fn.Name.Name], locations[fn.Name.Name] = fn, relative
			}
		}
	}
	var integer func(ast.Expr, int) (int, bool)
	integer = func(expr ast.Expr, id int) (int, bool) {
		switch value := expr.(type) {
		case *ast.Ident:
			return id, value.Name == "id"
		case *ast.BasicLit:
			if value.Kind == token.INT {
				n, err := strconv.ParseInt(value.Value, 0, 32)
				return int(n), err == nil
			}
		case *ast.ParenExpr:
			return integer(value.X, id)
		}
		return 0, false
	}
	var boolean func(ast.Expr, int) (bool, bool)
	boolean = func(expr ast.Expr, id int) (bool, bool) {
		switch value := expr.(type) {
		case *ast.Ident:
			if value.Name == "true" || value.Name == "false" {
				return value.Name == "true", true
			}
		case *ast.ParenExpr:
			return boolean(value.X, id)
		case *ast.BinaryExpr:
			if value.Op == token.LAND || value.Op == token.LOR {
				left, okL := boolean(value.X, id)
				right, okR := boolean(value.Y, id)
				if !okL || !okR {
					return false, false
				}
				if value.Op == token.LAND {
					return left && right, true
				}
				return left || right, true
			}
			left, okL := integer(value.X, id)
			right, okR := integer(value.Y, id)
			if !okL || !okR {
				return false, false
			}
			switch value.Op {
			case token.EQL:
				return left == right, true
			case token.NEQ:
				return left != right, true
			case token.LSS:
				return left < right, true
			case token.LEQ:
				return left <= right, true
			case token.GTR:
				return left > right, true
			case token.GEQ:
				return left >= right, true
			}
		}
		return false, false
	}
	var selectedCase func(*ast.SwitchStmt, int) *ast.CaseClause
	selectedCase = func(statement *ast.SwitchStmt, id int) *ast.CaseClause {
		var fallback *ast.CaseClause
		for _, item := range statement.Body.List {
			clause := item.(*ast.CaseClause)
			if clause.List == nil {
				fallback = clause
			}
			for _, expr := range clause.List {
				matches, known := false, false
				if statement.Tag == nil {
					matches, known = boolean(expr, id)
				} else {
					left, okL := integer(statement.Tag, id)
					right, okR := integer(expr, id)
					matches, known = left == right, okL && okR
				}
				must(known, "派送條件已變更，需人工核對")
				if matches {
					return clause
				}
			}
		}
		return fallback
	}
	gate := func(name string, id int) bool {
		fn := functions[name]
		must(fn != nil && len(fn.Body.List) == 1, "閘門結構已變更："+name)
		statement := fn.Body.List[0]
		if branch, ok := statement.(*ast.SwitchStmt); ok {
			clause := selectedCase(branch, id)
			must(clause != nil && len(clause.Body) == 1, "閘門分支結構已變更")
			statement = clause.Body[0]
		}
		ret, ok := statement.(*ast.ReturnStmt)
		must(ok && len(ret.Results) == 1, "閘門回傳結構已變更")
		value, known := boolean(ret.Results[0], id)
		must(known, "閘門表達式已變更")
		return value
	}
	confirm := functions["confirm"]
	must(confirm != nil, "缺正式confirm入口")
	var targetBody *ast.BlockStmt
	ast.Inspect(confirm.Body, func(node ast.Node) bool {
		if branch, ok := node.(*ast.IfStmt); ok {
			if selector, ok := branch.Cond.(*ast.SelectorExpr); ok && selector.Sel.Name == "nativeCommand0Targeting" {
				must(targetBody == nil, "正式指令分支重複")
				targetBody = branch.Body
				return false
			}
		}
		return true
	})
	must(targetBody != nil, "缺正式指令confirm分支")
	calls := func(id int) []string {
		var branch ast.Node
		for _, statement := range targetBody.List {
			switch statement := statement.(type) {
			case *ast.IfStmt:
				if matches, known := boolean(statement.Cond, id); known && matches {
					branch = statement.Body
				}
			case *ast.SwitchStmt:
				if branch == nil {
					branch = selectedCase(statement, id)
				}
			}
			if branch != nil {
				break
			}
		}
		must(branch != nil, "缺指令派送分支")
		found := map[string]bool{}
		var inspect func(ast.Node)
		inspect = func(root ast.Node) {
			if root == nil {
				return
			}
			ast.Inspect(root, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.FuncLit:
					return false // 完成callback不作進場呼叫證據。
				case *ast.IfStmt:
					if matches, known := boolean(node.Cond, id); known {
						if matches {
							inspect(node.Body)
						} else if node.Else != nil {
							inspect(node.Else)
						}
						return false
					}
				case *ast.CallExpr:
					if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
						name := selector.Sel.Name
						if strings.HasPrefix(name, "startNativeCommand") || name == "beginNativeCommand23Destination" || name == "ExecuteNativeCommandDerivedStrike" {
							found[name] = true
						}
					}
				}
				return true
			})
		}
		inspect(branch)
		result := make([]string, 0, len(found))
		for name := range found {
			result = append(result, name)
		}
		sort.Strings(result)
		return result
	}
	var spells []map[string]any
	must(json.Unmarshal(read("remake/assets/spells.json"), &spells) == nil && len(spells) == 36, "正式指令表不同")
	var entities struct {
		Names map[string]struct{ Name string } `json:"command_names"`
	}
	must(json.Unmarshal(read("remake/assets/locales/zh-Hant/entities.json"), &entities) == nil, "名稱表不同")
	var learn []struct {
		Index   int `json:"idx"`
		Entries []struct {
			Level int `json:"required_level"`
			ID    int `json:"command_id"`
		} `json:"entries"`
	}
	must(json.Unmarshal(read("remake/assets/data/command_learn.json"), &learn) == nil, "學習表不同")
	var rows []map[string]any
	var blocked, blockedLearned, stateOnly []int
	for id, spell := range spells {
		must(spell["id"] == float64(id), "指令表索引不同")
		accepted := gate("nativeCommandTargetSupported", id)
		var learning []map[string]int
		for _, group := range learn {
			for _, entry := range group.Entries {
				if entry.ID == id {
					learning = append(learning, map[string]int{"learn_idx": group.Index, "level": entry.Level})
				}
			}
		}
		if !accepted {
			blocked = append(blocked, id)
			if len(learning) > 0 {
				blockedLearned = append(blockedLearned, id)
			}
		}
		entryCalls := calls(id)
		state := len(entryCalls) == 1 && entryCalls[0] == "ExecuteNativeCommandDerivedStrike"
		if state {
			stateOnly = append(stateOnly, id)
		}
		rows = append(rows, map[string]any{"command": id, "name": entities.Names[strconv.Itoa(id)].Name,
			"player_menu_accepts": accepted, "player_confirm_calls": entryCalls, "player_state_only": state,
			"ai_dispatch_gate_accepts": gate("nativeAICommandHasIndexedOwner", id), "learn_table_entries": learning,
			"selection_mode": spell["dist"], "effect_mode": spell["range"], "target_code": spell["target"]})
	}
	read("tools/audit_fd2_battle_owners.go")
	report := map[string]any{"schema_version": 1, "status": "PASS-STATIC-COMMAND-ROUTE-AUDIT-WITH-GAPS",
		"scope":               "正式36筆指令、Go AST玩家選單閘門、confirm直接派送及AI派送閘門；學習表只證明資料列，不證明自然取得、動畫播放或完整玩家路徑。",
		"source_files_sha256": hashes, "commands": rows, "player_menu_blocked": blocked,
		"learn_table_commands_blocked_by_player_menu": blockedLearned, "player_state_only": stateOnly,
		"full_battle_hd_complete": false, "gpu_verified": false, "normal_player_verified": false,
		"locations": map[string]any{"player_gate": map[string]any{"file": locations["nativeCommandTargetSupported"], "line": set.Position(functions["nativeCommandTargetSupported"].Pos()).Line},
			"confirm": map[string]any{"file": locations["confirm"], "line": set.Position(targetBody.Pos()).Line},
			"ai_gate": map[string]any{"file": locations["nativeAICommandHasIndexedOwner"], "line": set.Position(functions["nativeAICommandHasIndexedOwner"].Pos()).Line}},
		"unverified": []string{"全部角色、側別、反擊資格與參數組合", "每個建構器的素材預檢、HD及音效完整性", "自我或空格範圍中心的正常確認", "全部正常玩家與敵方流程、保存與返回", "FIGANI408未知來源"}}
	data, err := json.MarshalIndent(report, "", "  ")
	must(err == nil, "JSON失敗")
	must(os.WriteFile(out, append(data, '\n'), 0644) == nil, "寫入失敗")
	fmt.Printf("指令36；玩家選單阻擋%v；其中學習表指令%v；只有狀態交易%v。這是靜態盤點，完整動畫未完成。\n", blocked, blockedLearned, stateOnly)
}
